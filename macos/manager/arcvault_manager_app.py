"""ArcVault Manager GUI for macOS Docker deployment.

This PyQt5 application wraps the shell scripts that control the ArcVault
Docker stack. It provides start/stop/update controls, health status, and
lightweight configuration editing for the `.env` file used by the server.

The app expects the packaged installer to stage helper scripts under
`~/Library/Application Support/ArcVault/bin`. These scripts are invoked to
interact with Docker so that all low-level logic lives in shell scripts that
mirror the existing Windows deployment.

The GUI is intentionally simple so that it can be packaged with PyInstaller
and embedded inside a macOS `.app` bundle.
"""

import json
import os
import shutil
import subprocess
import sys
import webbrowser
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

from PyQt5 import QtCore, QtGui, QtWidgets


APP_SUPPORT_ROOT = Path.home() / "Library" / "Application Support" / "ArcVault"
BIN_DIR = APP_SUPPORT_ROOT / "bin"
DOCKER_DIR = BIN_DIR / "docker"
CONFIG_DIR = APP_SUPPORT_ROOT / "config"
ENV_FILE = CONFIG_DIR / ".env"
LOG_DIR = APP_SUPPORT_ROOT / "#backup_logs"


class WorkerThread(QtCore.QThread):
    """Run a callable on a background thread and emit completion signals."""

    started = QtCore.pyqtSignal()
    finished = QtCore.pyqtSignal(int, str, str)

    def __init__(self, fn, *args, **kwargs):
        super().__init__()
        self.fn = fn
        self.args = args
        self.kwargs = kwargs

    def run(self):
        self.started.emit()
        try:
            code, out, err = self.fn(*self.args, **self.kwargs)
        except Exception as exc:  # pragma: no cover - defensive
            code, out, err = 1, "", str(exc)
        self.finished.emit(code, out, err)


def run_shell(script: Path, *extra_args: str, timeout: Optional[int] = None):
    """Execute a helper script and return (code, stdout, stderr)."""

    if not script.exists():
        return 1, "", f"Missing helper script: {script}"

    cmd = ["/bin/bash", str(script)] + list(extra_args)
    process = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=timeout,
        check=False,
    )
    return process.returncode, process.stdout, process.stderr


@dataclass
class StatusResult:
    health: str
    details: str


def read_status() -> StatusResult:
    script = DOCKER_DIR / "status.sh"
    code, out, err = run_shell(script)
    if code != 0:
        return StatusResult("Error", err.strip() or "Unable to read status")
    try:
        payload = json.loads(out)
    except json.JSONDecodeError:
        payload = {"health": "Unknown", "details": out.strip()}
    return StatusResult(payload.get("health", "Unknown"), payload.get("details", ""))


class ArcVaultManagerWindow(QtWidgets.QMainWindow):
    status_requested = QtCore.pyqtSignal()

    def __init__(self):
        super().__init__()
        self.setWindowTitle("ArcVault Manager")
        self.setMinimumSize(700, 480)

        self.status_label = QtWidgets.QLabel("Status: Unknown")
        self.status_label.setObjectName("statusLabel")

        self.refresh_button = QtWidgets.QPushButton("Refresh Status")
        self.refresh_button.clicked.connect(self.refresh_status)

        self.start_button = QtWidgets.QPushButton("Start")
        self.start_button.clicked.connect(lambda: self.run_action("start.sh", "Starting ArcVault..."))
        self.stop_button = QtWidgets.QPushButton("Stop")
        self.stop_button.clicked.connect(lambda: self.run_action("stop.sh", "Stopping ArcVault..."))
        self.restart_button = QtWidgets.QPushButton("Restart")
        self.restart_button.clicked.connect(lambda: self.run_action("restart.sh", "Restarting ArcVault..."))
        self.update_button = QtWidgets.QPushButton("Update")
        self.update_button.clicked.connect(lambda: self.run_action("update.sh", "Updating ArcVault images..."))

        self.open_ui_button = QtWidgets.QPushButton("Open Web UI")
        self.open_ui_button.clicked.connect(lambda: webbrowser.open("http://localhost:8501"))

        self.open_logs_button = QtWidgets.QPushButton("Open Logs Folder")
        self.open_logs_button.clicked.connect(self.open_logs)

        self.install_docker_button = QtWidgets.QPushButton("Install Docker Desktop")
        self.install_docker_button.clicked.connect(self.install_docker)

        self.log_output = QtWidgets.QPlainTextEdit()
        self.log_output.setReadOnly(True)

        self.env_editor = QtWidgets.QPlainTextEdit()
        self.env_editor.setPlaceholderText("Environment configuration (.env)")

        self.save_env_button = QtWidgets.QPushButton("Save Configuration")
        self.save_env_button.clicked.connect(self.save_env)

        button_layout = QtWidgets.QGridLayout()
        button_layout.addWidget(self.start_button, 0, 0)
        button_layout.addWidget(self.stop_button, 0, 1)
        button_layout.addWidget(self.restart_button, 0, 2)
        button_layout.addWidget(self.update_button, 0, 3)
        button_layout.addWidget(self.refresh_button, 1, 0, 1, 2)
        button_layout.addWidget(self.open_ui_button, 1, 2)
        button_layout.addWidget(self.open_logs_button, 1, 3)
        button_layout.addWidget(self.install_docker_button, 2, 0, 1, 4)

        control_widget = QtWidgets.QWidget()
        control_layout = QtWidgets.QVBoxLayout(control_widget)
        control_layout.addWidget(self.status_label)
        control_layout.addLayout(button_layout)

        splitter = QtWidgets.QSplitter()
        splitter.setOrientation(QtCore.Qt.Vertical)
        splitter.addWidget(self.log_output)

        config_group = QtWidgets.QGroupBox("Configuration")
        config_layout = QtWidgets.QVBoxLayout(config_group)
        config_layout.addWidget(self.env_editor)
        config_layout.addWidget(self.save_env_button)
        splitter.addWidget(config_group)

        central = QtWidgets.QWidget()
        layout = QtWidgets.QVBoxLayout(central)
        layout.addWidget(control_widget)
        layout.addWidget(splitter, 1)

        self.setCentralWidget(central)

        self.current_worker: Optional[WorkerThread] = None
        self._docker_warning_shown = False

        QtCore.QTimer.singleShot(100, self.load_env)
        QtCore.QTimer.singleShot(150, self.refresh_status)
        QtCore.QTimer.singleShot(200, self.update_control_states)

    # region helpers
    def append_log(self, message: str):
        cursor = self.log_output.textCursor()
        cursor.movePosition(QtGui.QTextCursor.End)
        cursor.insertText(message + "\n")
        self.log_output.setTextCursor(cursor)
        self.log_output.ensureCursorVisible()

    def set_buttons_enabled(self, enabled: bool):
        docker_present = self.docker_installed()
        for button in [
            self.start_button,
            self.stop_button,
            self.restart_button,
            self.update_button,
            self.refresh_button,
            self.open_ui_button,
        ]:
            button.setEnabled(enabled and docker_present)
        self.open_logs_button.setEnabled(enabled)
        self.save_env_button.setEnabled(enabled)
        self.install_docker_button.setEnabled(enabled)
        self.install_docker_button.setVisible(not docker_present)

    def docker_installed(self) -> bool:
        if shutil.which("docker"):
            return True
        return Path("/Applications/Docker.app").exists()

    def update_control_states(self):
        docker_present = self.docker_installed()
        self.install_docker_button.setVisible(not docker_present)
        if docker_present:
            self._docker_warning_shown = False
        elif not self._docker_warning_shown:
            self.append_log(
                "Docker Desktop not detected. Click 'Install Docker Desktop' to download and install the latest build."
            )
            self.status_label.setText("Status: Docker not installed")
            self._docker_warning_shown = True

    def install_docker(self):
        script = BIN_DIR / "install_docker.sh"
        self.append_log("Downloading and installing Docker Desktop...")
        self.execute_worker(run_shell, args=(script,), callback=self.on_docker_install_finished)

    def on_docker_install_finished(self, out: str):
        self.update_control_states()
        self.refresh_status()

    def run_action(self, script_name: str, description: str):
        script = DOCKER_DIR / script_name
        self.append_log(description)
        self.execute_worker(run_shell, args=(script,))

    def execute_worker(self, fn, args=(), kwargs=None, callback=None):
        if kwargs is None:
            kwargs = {}
        if self.current_worker and self.current_worker.isRunning():
            self.append_log("An operation is already in progress.")
            return

        self.set_buttons_enabled(False)
        self.current_worker = WorkerThread(fn, *args, **kwargs)
        self.current_worker.started.connect(lambda: self.status_label.setText("Status: Working..."))
        self.current_worker.finished.connect(lambda code, out, err: self.on_worker_finished(code, out, err, callback))
        self.current_worker.start()

    def on_worker_finished(self, code: int, out: str, err: str, callback):
        self.set_buttons_enabled(True)
        if code == 0:
            if callback:
                callback(out)
            if out.strip():
                self.append_log(out.strip())
        else:
            self.append_log(f"Error (code {code}): {err.strip() or out.strip() or 'Unknown error'}")
        if callback is None:
            self.refresh_status()
        self.update_control_states()

    def refresh_status(self):
        if not self.docker_installed():
            self.status_label.setText("Status: Docker not installed")
            self.update_control_states()
            return

        def do_status():
            status = read_status()
            payload = json.dumps(status.__dict__)
            return 0, payload, ""

        def handle_status(out: str):
            try:
                payload = json.loads(out)
            except json.JSONDecodeError:
                payload = {"health": "Unknown", "details": out}
            self.status_label.setText(f"Status: {payload.get('health', 'Unknown')}")
            details = payload.get("details")
            if details:
                formatted = self.format_details(details)
                if formatted:
                    self.append_log(formatted)

        self.execute_worker(do_status, callback=handle_status)

    def load_env(self):
        try:
            content = ENV_FILE.read_text(encoding="utf-8")
        except FileNotFoundError:
            content = ""
        except OSError as exc:
            self.append_log(f"Failed to read {ENV_FILE}: {exc}")
            content = ""
        self.env_editor.setPlainText(content)

    def save_env(self):
        CONFIG_DIR.mkdir(parents=True, exist_ok=True)
        LOG_DIR.mkdir(parents=True, exist_ok=True)
        try:
            ENV_FILE.write_text(self.env_editor.toPlainText(), encoding="utf-8")
            self.append_log("Configuration saved.")
        except OSError as exc:
            self.append_log(f"Failed to write configuration: {exc}")

    def open_logs(self):
        LOG_DIR.mkdir(parents=True, exist_ok=True)
        subprocess.run(["open", str(LOG_DIR)])

    # endregion

    def format_details(self, details) -> str:
        if isinstance(details, str):
            return details
        if isinstance(details, list):
            lines = []
            for item in details:
                if not isinstance(item, dict):
                    lines.append(str(item))
                    continue
                service = item.get("service", "unknown")
                state = item.get("state", "")
                health = item.get("health", "")
                lines.append(f"{service}: state={state or 'n/a'}, health={health or 'n/a'}")
            return "\n".join(lines)
        return json.dumps(details)


def main():
    app = QtWidgets.QApplication(sys.argv)
    window = ArcVaultManagerWindow()
    window.show()
    sys.exit(app.exec_())


if __name__ == "__main__":
    main()
