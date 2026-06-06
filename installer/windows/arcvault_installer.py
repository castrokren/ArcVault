#!/usr/bin/env python3
"""
ArcVault Windows Installer
Creates a native Windows installer for ArcVault (Coordinator, Agent, or Both)
Can be compiled to .exe using PyInstaller
"""

import os
import sys
import json
import shutil
import subprocess
import time
import secrets
import webbrowser
from pathlib import Path
import tkinter as tk
from tkinter import ttk, messagebox
import threading


class ArcVaultInstaller:
    """Main installer class"""

    # Install locations — must match what coordinator/agent read at runtime.
    # coordinator.exe reads config.json from its own directory (os.Executable()).
    # agent.exe reads agent-config.yaml from its own directory.
    COORD_DIR = Path("C:/ArcVault")
    AGENT_DIR = Path("C:/ArcVault-Agent")

    def __init__(self):
        self.version = "0.3.0"
        self.components = set()
        self.coordinator_port = 8080
        self.admin_token = ""
        self.agent_id = ""
        self.agent_token = ""
        self.coordinator_url = ""
        self.credential_key = ""
        self.root = None

    def _setup_styles(self):
        """Configure ttk styles with the clam theme for reliable color rendering."""
        style = ttk.Style()
        # clam theme respects background/foreground; vista/winnative ignore them
        try:
            style.theme_use("clam")
        except Exception:
            pass

        style.configure(
            "Primary.TButton",
            background="#0078d4",
            foreground="white",
            font=("Segoe UI", 10),
            padding=(16, 6),
            borderwidth=0,
            focusthickness=0,
        )
        style.map(
            "Primary.TButton",
            background=[("active", "#005fa3"), ("pressed", "#004e8c")],
            foreground=[("active", "white")],
        )

        style.configure(
            "Secondary.TButton",
            background="#e1e1e1",
            foreground="#1a1a1a",
            font=("Segoe UI", 10),
            padding=(16, 6),
            borderwidth=0,
            focusthickness=0,
        )
        style.map(
            "Secondary.TButton",
            background=[("active", "#c8c8c8"), ("pressed", "#b0b0b0")],
            foreground=[("active", "#1a1a1a")],
        )

    def run(self):
        """Start the installer GUI"""
        self.root = tk.Tk()
        self.root.title(f"ArcVault {self.version} Setup Wizard")
        self.root.geometry("600x420")
        self.root.resizable(False, False)
        self._setup_styles()
        self.show_welcome()
        self.root.mainloop()

    # ------------------------------------------------------------------
    # Helper
    # ------------------------------------------------------------------

    def _nav_buttons(self, parent, next_text, next_cmd, back_cmd=None):
        """Render consistent Next/Back/Cancel buttons."""
        bf = ttk.Frame(parent)
        bf.pack(fill=tk.X, pady=(20, 0))
        ttk.Button(bf, text=next_text, style="Primary.TButton",
                   command=next_cmd).pack(side=tk.RIGHT)
        if back_cmd:
            ttk.Button(bf, text="< Back", style="Secondary.TButton",
                       command=back_cmd).pack(side=tk.LEFT)
        else:
            ttk.Button(bf, text="Cancel", style="Secondary.TButton",
                       command=self.root.quit).pack(side=tk.LEFT)

    # ------------------------------------------------------------------
    # Screens
    # ------------------------------------------------------------------

    def show_welcome(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Welcome to ArcVault",
                  font=("Arial", 20, "bold")).pack(pady=(10, 4))
        ttk.Label(frame, text=f"Version {self.version}",
                  font=("Arial", 11)).pack()
        ttk.Label(frame,
                  text="This wizard will guide you through installing ArcVault\n"
                       "on your Windows machine.",
                  justify=tk.CENTER, font=("Arial", 10)).pack(pady=16)
        ttk.Label(frame,
                  text="You can choose to install:\n"
                       "  • Coordinator (server)\n"
                       "  • Agent (client)\n"
                       "  • Both",
                  justify=tk.LEFT, font=("Arial", 10)).pack(pady=8)
        self._nav_buttons(frame, "Next >", self.show_component_selection)

    def show_component_selection(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Select Components",
                  font=("Arial", 16, "bold")).pack(pady=(10, 16))

        coordinator_var = tk.BooleanVar()
        ttk.Checkbutton(frame,
                        text="Coordinator (Server) — Central backup orchestration",
                        variable=coordinator_var).pack(anchor=tk.W, pady=6)

        agent_var = tk.BooleanVar()
        ttk.Checkbutton(frame,
                        text="Agent (Client) — Backup execution on this machine",
                        variable=agent_var).pack(anchor=tk.W, pady=6)

        def next_step():
            if not coordinator_var.get() and not agent_var.get():
                messagebox.showerror("Selection Required",
                                     "Please select at least one component")
                return
            self.components = set()
            if coordinator_var.get():
                self.components.add("coordinator")
            if agent_var.get():
                self.components.add("agent")
            if "coordinator" in self.components:
                self.show_coordinator_config()
            else:
                self.show_agent_config()

        self._nav_buttons(frame, "Next >", next_step, self.show_welcome)

    def show_coordinator_config(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Coordinator Configuration",
                  font=("Arial", 16, "bold")).pack(pady=(10, 16))

        def row(lbl, **kw):
            r = ttk.Frame(frame)
            r.pack(fill=tk.X, pady=5)
            ttk.Label(r, text=lbl, width=18, anchor=tk.W).pack(side=tk.LEFT)
            e = ttk.Entry(r, **kw)
            e.pack(side=tk.LEFT, padx=8, fill=tk.X, expand=True)
            return e

        port_var = row("Port:", width=10)
        port_var.insert(0, "8080")
        ttk.Label(frame, text="Default login: admin / changeme  (you'll be prompted to change it on first login)",
                  font=("Segoe UI", 9), foreground="#555555").pack(anchor=tk.W, pady=(4, 0))

        def next_step():
            try:
                self.coordinator_port = int(port_var.get())
            except ValueError:
                messagebox.showerror("Invalid", "Port must be a number")
                return
            self.admin_token = self.generate_token(32)
            if "agent" in self.components:
                self.show_agent_config()
            else:
                self.show_review()

        self._nav_buttons(frame, "Next >", next_step,
                          self.show_component_selection)

    def show_agent_config(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Agent Configuration",
                  font=("Arial", 16, "bold")).pack(pady=(10, 16))

        def row(lbl, **kw):
            r = ttk.Frame(frame)
            r.pack(fill=tk.X, pady=5)
            ttk.Label(r, text=lbl, width=18, anchor=tk.W).pack(side=tk.LEFT)
            e = ttk.Entry(r, **kw)
            e.pack(side=tk.LEFT, padx=8, fill=tk.X, expand=True)
            return e

        if "coordinator" in self.components:
            default_url = f"http://localhost:{self.coordinator_port}"
            self.coordinator_url = default_url
            url_var = row("Coordinator URL:", width=36)
            url_var.insert(0, default_url)
            url_var.config(state="readonly")
        else:
            url_var = row("Coordinator URL:", width=36)

        id_var = row("Agent ID:", width=24)
        id_var.insert(0, os.environ.get("COMPUTERNAME", "agent-1"))

        if "coordinator" in self.components:
            # Reuse the coordinator's admin_token — authMiddleware always accepts it,
            # and the DB is empty on a fresh install so a separate agent token would
            # be rejected (401 → registration failure → Error 1067 on service start).
            self.agent_token = self.admin_token
            token_var = row("Auth Token:", width=36)
            token_var.insert(0, self.agent_token[:16] + "...")
            token_var.config(state="readonly")
        else:
            token_var = row("Auth Token:", width=36)

        def next_step():
            if "coordinator" not in self.components:
                if not url_var.get():
                    messagebox.showerror("Required", "Coordinator URL is required")
                    return
                self.coordinator_url = url_var.get()
                if not token_var.get():
                    messagebox.showerror("Required", "Auth token is required")
                    return
                self.agent_token = token_var.get()
            self.agent_id = id_var.get()
            self.show_review()

        self._nav_buttons(frame, "Next >", next_step,
                          self.show_component_selection)

    def show_review(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Review Configuration",
                  font=("Arial", 16, "bold")).pack(pady=(10, 16))

        lines = [f"Components:  {', '.join(c.title() for c in sorted(self.components))}"]
        if "coordinator" in self.components:
            lines += [f"Port:  {self.coordinator_port}",
                      f"Install dir:  {self.COORD_DIR}",
                      f"Default login:  admin / changeme"]
        if "agent" in self.components:
            lines += [f"Coordinator URL:  {self.coordinator_url}",
                      f"Agent ID:  {self.agent_id}"]

        ttk.Label(frame, text="\n".join(lines),
                  justify=tk.LEFT, font=("Arial", 10)).pack(pady=8, anchor=tk.W)
        ttk.Label(frame, text="Click Install to begin.",
                  font=("Arial", 10, "italic")).pack(pady=8)

        self._nav_buttons(frame, "Install", self.install,
                          self.show_agent_config if "agent" in self.components
                          else self.show_coordinator_config)

    def install(self):
        self.clear_window()
        frame = ttk.Frame(self.root, padding=20)
        frame.pack(fill=tk.BOTH, expand=True)

        ttk.Label(frame, text="Installing ArcVault...",
                  font=("Arial", 16, "bold")).pack(pady=(10, 16))

        progress = ttk.Progressbar(frame, length=460, mode="indeterminate")
        progress.pack(pady=10)
        progress.start()

        status = ttk.Label(frame, text="Preparing...", font=("Arial", 10))
        status.pack(pady=8)

        def do_install():
            try:
                status.config(text="Writing configuration...")
                self.root.update()

                if "coordinator" in self.components:
                    self.write_coordinator_config()
                if "agent" in self.components:
                    self.write_agent_config()

                status.config(text="Installing services...")
                self.root.update()

                if "coordinator" in self.components:
                    self.install_service("coordinator")
                if "agent" in self.components:
                    self.install_service("agent")

                if "coordinator" in self.components:
                    # Wait for coordinator to start before opening dashboard
                    status.config(text="Waiting for coordinator to start...")
                    self.root.update()
                    time.sleep(3)
                    status.config(text="Opening dashboard...")
                    self.root.update()
                    self.open_browser(f"http://localhost:{self.coordinator_port}")

                progress.stop()
                messagebox.showinfo(
                    "Installation Complete",
                    "ArcVault has been installed successfully!\n\n"
                    "The services are starting and the dashboard will\n"
                    "open in your browser momentarily."
                )
                self.root.quit()

            except Exception as exc:
                progress.stop()
                messagebox.showerror("Installation Error", str(exc))
                self.show_review()

        threading.Thread(target=do_install, daemon=True).start()

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def generate_credential_key(self):
        """Generate a 32-byte credential encryption key in hex format."""
        return secrets.token_hex(32)

    def get_or_create_credential_key(self):
        """
        Get existing credential key from config.json or generate new one.
        Returns (key, is_existing).
        """
        try:
            config_path = self.COORD_DIR / "config.json"
            if config_path.exists():
                with open(config_path) as f:
                    existing = json.load(f)
                key = existing.get("credential_key", "")
                if key and len(key) == 64:
                    return key, True
        except Exception:
            pass

        # Generate new key if not found
        return self.generate_credential_key(), False

    def set_service_environment_variable(self, service_name: str, var_name: str, var_value: str):
        """Set an environment variable for a Windows service via Registry."""
        try:
            reg_path = f'HKLM\\SYSTEM\\CurrentControlSet\\Services\\{service_name}\\Environment'
            subprocess.run(
                ['reg', 'add', reg_path, '/v', var_name, '/d', var_value, '/f'],
                capture_output=True, text=True, shell=True, check=True
            )
        except Exception as e:
            print(f"Warning: Failed to set service environment variable: {e}")

    def write_coordinator_config(self):
        """Write config.json next to coordinator.exe — that's where it looks at runtime."""
        self.COORD_DIR.mkdir(parents=True, exist_ok=True)

        # Generate or retrieve credential key before writing config
        key, is_existing = self.get_or_create_credential_key()
        self.credential_key = key

        config = {
            "port": self.coordinator_port,
            "admin_token": self.admin_token,
            "database_path": str(self.COORD_DIR / "arcvault.db"),
            "credential_key": key,
            "environment": "production",
        }
        with open(self.COORD_DIR / "config.json", "w") as f:
            json.dump(config, f, indent=2)

        # If new key was generated, display it to user (once)
        if not is_existing:
            messagebox.showinfo(
                "Credential Key Generated",
                f"A credential encryption key has been generated:\n\n{key}\n\n"
                "⚠️ Save this key in a secure location!\n"
                "You will need it to re-install or migrate the system.\n\n"
                "The key is stored in C:\\ArcVault\\config.json."
            )

    def write_agent_config(self):
        """Write agent-config.yaml next to agent.exe — that's where it looks at runtime."""
        self.AGENT_DIR.mkdir(parents=True, exist_ok=True)
        content = (
            f"agent_id: {self.agent_id}\n"
            f"coordinator_url: {self.coordinator_url}\n"
            f"auth_token: {self.agent_token}\n"
        )
        with open(self.AGENT_DIR / "agent-config.yaml", "w") as f:
            f.write(content)

    def install_service(self, service_type: str):
        """
        Copy the bundled binary to the install directory, then use the binary's
        own install-service command.  This ensures the service is registered with
        the 'run-service' argument that Windows SCM requires, and uses the correct
        service name (arcvault-coordinator / arcvault-agent).
        """
        try:
            base_path = Path(sys._MEIPASS) if getattr(sys, "frozen", False) else Path.cwd()

            names = {
                "coordinator": ("coordinator.exe", "arcvault-coordinator", self.COORD_DIR),
                "agent":       ("agent.exe",       "arcvault-agent",       self.AGENT_DIR),
            }
            if service_type not in names:
                raise Exception(f"Unknown service type: {service_type}")
            binary_name, service_name, install_dir = names[service_type]

            binary_src = base_path / binary_name
            if not binary_src.exists():
                raise Exception(f"Binary not found: {binary_src}")

            # Prepare install directory
            install_dir.mkdir(parents=True, exist_ok=True)

            # Stop and remove any existing service registration.
            # Use 'sc' directly — doesn't require the binary to be present,
            # so it works on both fresh installs and reinstalls.
            subprocess.run(["sc", "stop", service_name],
                           capture_output=True, text=True, shell=True)
            time.sleep(1)
            subprocess.run(["sc", "delete", service_name],
                           capture_output=True, text=True, shell=True)
            # Wait until SCM fully releases the registration (up to 15 s).
            # We must see exit code 1060 (ERROR_SERVICE_DOES_NOT_EXIST) — NOT just
            # any non-zero code. Exit code 1072 means "marked for deletion" (still
            # alive); creating a new service while that's true causes start failures.
            for _ in range(30):
                chk = subprocess.run(
                    ["sc", "query", service_name],
                    capture_output=True, text=True, shell=True,
                )
                gone = (
                    chk.returncode == 1060
                    or "does not exist" in chk.stdout.lower()
                    or "does not exist" in chk.stderr.lower()
                )
                if gone:
                    break
                time.sleep(0.5)
            # Extra buffer: give SCM time to flush the registry entry
            time.sleep(2)

            # Copy binary to install directory
            binary_dst = install_dir / binary_name
            shutil.copy(binary_src, binary_dst)

            # Register service using the binary's own install-service command.
            # This correctly passes 'run-service' to SCM (plain 'sc create binPath='
            # would start without arguments and exit immediately with code 1).
            result = subprocess.run(
                [str(binary_dst), "install-service"],
                capture_output=True, text=True, shell=True,
            )
            if result.returncode != 0:
                raise Exception(f"install-service failed: {result.stdout} {result.stderr}")

            # Set credential key environment variable for coordinator service
            if service_type == "coordinator" and self.credential_key:
                self.set_service_environment_variable("arcvault-coordinator",
                                                       "ARCVAULT_CREDENTIAL_KEY",
                                                       self.credential_key)

            start = subprocess.run(["sc", "start", service_name],
                                   capture_output=True, text=True, shell=True)
            if start.returncode != 0 and "already running" not in start.stderr.lower():
                raise Exception(f"sc start failed: {start.stderr}")

        except Exception as exc:
            raise Exception(f"Service install failed ({service_type}): {exc}")

    def open_browser(self, url: str):
        try:
            webbrowser.open(url)
        except Exception:
            pass

    def generate_token(self, length: int = 32) -> str:
        return secrets.token_hex(length)

    def clear_window(self):
        for w in self.root.winfo_children():
            w.destroy()


if __name__ == "__main__":
    ArcVaultInstaller().run()
