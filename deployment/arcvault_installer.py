#!/usr/bin/env python3
"""
ArcVault 2.0 — Automated Installer
Handles downloading, initializing, and installing ArcVault as a Windows service.
Requests UAC elevation at startup so service install works without extra steps.
"""

import ctypes
import sys
import os
import subprocess
import threading
import queue
import re
import time
import webbrowser
import urllib.request
import shutil
import tkinter as tk
from tkinter import messagebox

# ── Admin elevation ────────────────────────────────────────────────────────────

def is_admin():
    try:
        return ctypes.windll.shell32.IsUserAnAdmin()
    except Exception:
        return False

def elevate_and_restart():
    """Re-launch this exe via UAC (runas) and exit the current process."""
    script = os.path.abspath(sys.argv[0])
    params = " ".join(f'"{a}"' for a in sys.argv[1:])
    ctypes.windll.shell32.ShellExecuteW(
        None, "runas", sys.executable, f'"{script}" {params}', None, 1
    )
    sys.exit(0)

# ── Constants ─────────────────────────────────────────────────────────────────

VERSION    = "v1.0.0"
GITHUB     = "https://github.com/castrokren/ArcVault/releases/latest/download"
COORD_URL  = f"{GITHUB}/coordinator-windows-amd64.exe"
AGENT_URL  = f"{GITHUB}/agent-windows-amd64.exe"
COORD_DIR  = r"C:\ArcVault"
AGENT_DIR  = r"C:\ArcVault-Agent"
COORD_BIN  = os.path.join(COORD_DIR, "coordinator.exe")
AGENT_BIN  = os.path.join(AGENT_DIR, "agent.exe")
AGENT_CFG  = os.path.join(AGENT_DIR, "agent-config.yaml")
COORD_SVC  = "arcvault-coordinator"
AGENT_SVC  = "arcvault-agent"
DASHBOARD  = "http://localhost:8080"

# ── Palette ───────────────────────────────────────────────────────────────────

BG       = "#0f1117"
BG2      = "#161b27"
BG3      = "#1e2235"
ACCENT   = "#6366f1"
ACCENT_H = "#818cf8"
SUCCESS  = "#22c55e"
DANGER   = "#ef4444"
TEXT     = "#e2e8f0"
MUTED    = "#94a3b8"
BORDER   = "#2d3748"
LOG_BG   = "#0d1117"
LOG_FG   = "#94a3b8"
WARN_BG  = "#1c1a0f"
WARN_FG  = "#fbbf24"

F_SMALL  = ("Segoe UI", 9)
F_BODY   = ("Segoe UI", 10)
F_TITLE  = ("Segoe UI", 14, "bold")
F_SUB    = ("Segoe UI", 11)
F_BTN    = ("Segoe UI", 10, "bold")
F_MONO   = ("Consolas", 9)
F_TOKEN  = ("Consolas", 11, "bold")


# ── Install Runner ─────────────────────────────────────────────────────────────

class InstallRunner:
    """
    Runs install steps in a background thread.
    Communicates with the UI via a thread-safe queue.
    """

    def __init__(self, component, agent_cfg=None):
        self.component  = component   # 'coordinator' | 'agent'
        self.agent_cfg  = agent_cfg   # dict: coordinator_url, agent_name, auth_token
        self.q          = queue.Queue()
        self.admin_token = None

    # ── Queue helpers ──────────────────────────────────────────────────────────

    def _emit(self, event, **kw):
        self.q.put({"event": event, **kw})

    def _log(self, text, style="normal"):
        self._emit("log", text=text, style=style)

    def _step_start(self, index, name):
        self._emit("step_start", index=index, name=name)

    def _step_done(self, index):
        self._emit("step_done", index=index)

    def _step_fail(self, index, message):
        self._emit("step_fail", index=index, message=message)

    # ── Execution helpers ──────────────────────────────────────────────────────

    def _run_cmd(self, args, cwd=None):
        """Run a command; stream every output line to the log. Returns (rc, output)."""
        self._log(f"› {' '.join(args)}", style="cmd")
        proc = subprocess.Popen(
            args,
            cwd=cwd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            creationflags=subprocess.CREATE_NO_WINDOW,
        )
        lines = []
        for line in proc.stdout:
            line = line.rstrip()
            if line:
                self._log(line)
                lines.append(line)
        proc.wait()
        return proc.returncode, "\n".join(lines)

    def _download(self, url, dest):
        self._log(f"Downloading from GitHub…", style="info")
        tmp = dest + ".part"

        def reporthook(count, block_size, total_size):
            if total_size > 0:
                pct = min(count * block_size * 100 / total_size, 100)
                self._emit("progress", pct=pct)

        urllib.request.urlretrieve(url, tmp, reporthook)
        shutil.move(tmp, dest)
        self._emit("progress", pct=100)
        self._log(f"Saved: {dest}", style="ok")

    def _cleanup_service(self, svc_name):
        """Stop and delete a service if it already exists. Ignores errors silently."""
        result = subprocess.run(
            ["sc", "query", svc_name],
            capture_output=True, text=True,
            creationflags=subprocess.CREATE_NO_WINDOW,
        )
        if "does not exist" in result.stdout or result.returncode != 0:
            return  # nothing to clean up
        self._log(f"Removing existing {svc_name} registration…", style="info")
        subprocess.run(["sc", "stop", svc_name],
                       capture_output=True, creationflags=subprocess.CREATE_NO_WINDOW)
        time.sleep(1)
        subprocess.run(["sc", "delete", svc_name],
                       capture_output=True, creationflags=subprocess.CREATE_NO_WINDOW)
        # Wait for SCM to release the registration (up to 5 s)
        for _ in range(10):
            chk = subprocess.run(
                ["sc", "query", svc_name],
                capture_output=True, text=True,
                creationflags=subprocess.CREATE_NO_WINDOW,
            )
            if "does not exist" in chk.stdout or chk.returncode != 0:
                break
            time.sleep(0.5)

    def _service_running(self, svc_name):
        result = subprocess.run(
            ["sc", "query", svc_name],
            capture_output=True,
            text=True,
            creationflags=subprocess.CREATE_NO_WINDOW,
        )
        return "RUNNING" in result.stdout

    # ── Main entry ────────────────────────────────────────────────────────────

    def start(self):
        threading.Thread(target=self._run, daemon=True).start()

    def _run(self):
        try:
            if self.component == "coordinator":
                self._install_coordinator()
            else:
                self._install_agent()
        except Exception as exc:
            self._emit("fatal", message=str(exc))

    # ── Coordinator install ────────────────────────────────────────────────────

    def _install_coordinator(self):
        steps = [
            "Create install directory",
            "Download coordinator binary",
            "Initialize coordinator",
            "Install Windows service",
            "Start & verify service",
        ]
        self._emit("steps", steps=steps)

        # Step 0 — directory
        self._step_start(0, steps[0])
        try:
            os.makedirs(COORD_DIR, exist_ok=True)
            self._log(f"Ready: {COORD_DIR}", style="ok")
            self._step_done(0)
        except Exception as exc:
            self._step_fail(0, str(exc))
            return

        # Step 1 — download
        self._step_start(1, steps[1])
        try:
            self._download(COORD_URL, COORD_BIN)
            self._step_done(1)
        except Exception as exc:
            self._step_fail(1, f"Download failed: {exc}")
            return

        # Step 2 — init
        # Pipe answers so config/db stay in COORD_DIR (not the user home dir).
        # The coordinator prompts for: (1) port, (2) db path.
        # Sending blank accepts the default port; db path is set to COORD_DIR.
        self._step_start(2, steps[2])
        self._log(f"› {COORD_BIN} init", style="cmd")
        db_path   = os.path.join(COORD_DIR, "arcvault.db")
        init_input = f"\n{db_path}\n"
        try:
            proc = subprocess.Popen(
                [COORD_BIN, "init"],
                cwd=COORD_DIR,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                creationflags=subprocess.CREATE_NO_WINDOW,
            )
            output, _ = proc.communicate(input=init_input, timeout=30)
        except subprocess.TimeoutExpired:
            proc.kill()
            self._step_fail(2, "Init timed out.")
            return
        for line in output.splitlines():
            if line.strip():
                self._log(line)
        if proc.returncode != 0:
            self._step_fail(2, f"Init failed (exit {proc.returncode})")
            return
        match = re.search(r"Admin token[^:]*:\s*(\S+)", output)
        if match:
            self.admin_token = match.group(1)
            self._emit("token", token=self.admin_token)
        self._step_done(2)

        # Step 3 — install service
        self._step_start(3, steps[3])
        # Clean up any leftover registration from a previous failed install
        self._cleanup_service(COORD_SVC)
        rc, _ = self._run_cmd([COORD_BIN, "install-service"], cwd=COORD_DIR)
        if rc != 0:
            self._step_fail(3, "Service install failed. Confirm you ran as Administrator.")
            return
        self._step_done(3)

        # Step 4 — start & verify
        self._step_start(4, steps[4])
        self._run_cmd(["sc", "start", COORD_SVC])
        time.sleep(2)
        if self._service_running(COORD_SVC):
            self._log("Service is running.", style="ok")
            self._step_done(4)
            self._emit("done")
        else:
            self._step_fail(4, "Service did not start. Check Windows Event Viewer → Application.")

    # ── Agent install ──────────────────────────────────────────────────────────

    def _install_agent(self):
        cfg   = self.agent_cfg
        steps = [
            "Create install directory",
            "Download agent binary",
            "Write agent config",
            "Install Windows service",
            "Start & verify service",
        ]
        self._emit("steps", steps=steps)

        # Step 0 — directory
        self._step_start(0, steps[0])
        try:
            os.makedirs(AGENT_DIR, exist_ok=True)
            self._log(f"Ready: {AGENT_DIR}", style="ok")
            self._step_done(0)
        except Exception as exc:
            self._step_fail(0, str(exc))
            return

        # Step 1 — download
        self._step_start(1, steps[1])
        try:
            self._download(AGENT_URL, AGENT_BIN)
            self._step_done(1)
        except Exception as exc:
            self._step_fail(1, f"Download failed: {exc}")
            return

        # Step 2 — write config
        self._step_start(2, steps[2])
        try:
            content = (
                f"coordinator_url: {cfg['coordinator_url']}\n"
                f"auth_token: {cfg['auth_token']}\n"
                f"agent_id: {cfg['agent_name']}\n"
            )
            with open(AGENT_CFG, "w") as fh:
                fh.write(content)
            self._log(f"Config written: {AGENT_CFG}", style="ok")
            self._step_done(2)
        except Exception as exc:
            self._step_fail(2, str(exc))
            return

        # Step 3 — install service
        self._step_start(3, steps[3])
        self._cleanup_service(AGENT_SVC)
        rc, _ = self._run_cmd(
            [AGENT_BIN, "install-service", "-config", AGENT_CFG],
            cwd=AGENT_DIR,
        )
        if rc != 0:
            self._step_fail(3, "Service install failed. Confirm you ran as Administrator.")
            return
        self._step_done(3)

        # Step 4 — start & verify
        self._step_start(4, steps[4])
        self._run_cmd(["sc", "start", AGENT_SVC])
        time.sleep(2)
        if self._service_running(AGENT_SVC):
            self._log("Agent service is running.", style="ok")
            self._step_done(4)
            self._emit("done")
        else:
            self._step_fail(4, "Service did not start. Check Windows Event Viewer → Application.")


# ── GUI Widgets ────────────────────────────────────────────────────────────────

class StepList(tk.Frame):
    """Left-panel step list with status icons."""

    _ICONS = {
        "pending": ("○", MUTED),
        "running": ("◎", ACCENT),
        "done":    ("✓", SUCCESS),
        "fail":    ("✗", DANGER),
    }

    def __init__(self, parent, **kw):
        super().__init__(parent, bg=BG3, **kw)
        self._rows = []

    def load(self, steps):
        for w in self.winfo_children():
            w.destroy()
        self._rows = []
        for name in steps:
            row = tk.Frame(self, bg=BG3, padx=14, pady=9)
            row.pack(fill="x")
            icon = tk.Label(row, text="○", font=("Segoe UI", 11),
                            bg=BG3, fg=MUTED, width=2, anchor="w")
            icon.pack(side="left", padx=(0, 8))
            label = tk.Label(row, text=name, font=F_BODY,
                             bg=BG3, fg=MUTED, anchor="w", wraplength=160, justify="left")
            label.pack(side="left", fill="x", expand=True)
            self._rows.append((icon, label))

    def set_status(self, index, status):
        if index >= len(self._rows):
            return
        icon_lbl, name_lbl = self._rows[index]
        char, color = self._ICONS.get(status, ("○", MUTED))
        icon_lbl.config(text=char, fg=color)
        name_lbl.config(fg=TEXT if status in ("running", "done") else (DANGER if status == "fail" else MUTED))


class LogView(tk.Frame):
    """Scrollable monospace log area with coloured output styles."""

    def __init__(self, parent, **kw):
        super().__init__(parent, bg=LOG_BG, **kw)
        txt = tk.Text(
            self, font=F_MONO, bg=LOG_BG, fg=LOG_FG,
            relief="flat", bd=0, padx=12, pady=10,
            wrap="none", state="disabled", cursor="arrow",
        )
        sb = tk.Scrollbar(self, orient="vertical", command=txt.yview,
                          bg=BG2, troughcolor=LOG_BG)
        txt.configure(yscrollcommand=sb.set)
        txt.tag_configure("ok",     foreground=SUCCESS)
        txt.tag_configure("info",   foreground=ACCENT_H)
        txt.tag_configure("cmd",    foreground="#7ee787")
        txt.tag_configure("error",  foreground=DANGER)
        txt.tag_configure("normal", foreground=LOG_FG)
        sb.pack(side="right", fill="y")
        txt.pack(side="left", fill="both", expand=True)
        self._txt = txt

    def append(self, text, style="normal"):
        self._txt.config(state="normal")
        self._txt.insert("end", text + "\n", style)
        self._txt.see("end")
        self._txt.config(state="disabled")

    def clear(self):
        self._txt.config(state="normal")
        self._txt.delete("1.0", "end")
        self._txt.config(state="disabled")


# ── Main Application ───────────────────────────────────────────────────────────

class ArcVaultInstaller(tk.Tk):

    def __init__(self):
        super().__init__()
        self.title("ArcVault 2.0 — Installer")
        self.geometry("900x650")
        self.minsize(800, 580)
        self.configure(bg=BG)
        self.resizable(True, True)

        # Center window
        self.update_idletasks()
        sw, sh = self.winfo_screenwidth(), self.winfo_screenheight()
        self.geometry(f"900x650+{(sw - 900) // 2}+{(sh - 650) // 2}")

        self.component   = None
        self.runner      = None
        self.admin_token = None
        self._poll_id    = None

        self._build_chrome()
        self._show_welcome()

    # ── Chrome ─────────────────────────────────────────────────────────────────

    def _build_chrome(self):
        # Header
        hdr = tk.Frame(self, bg=BG2, height=56)
        hdr.pack(fill="x")
        hdr.pack_propagate(False)
        brand = tk.Frame(hdr, bg=BG2)
        brand.pack(side="left", padx=22, pady=10)
        tk.Label(brand, text="⬡", font=("Segoe UI", 20), bg=BG2, fg=ACCENT).pack(side="left", padx=(0, 8))
        tk.Label(brand, text="ArcVault", font=("Segoe UI", 14, "bold"), bg=BG2, fg=TEXT).pack(side="left")
        tk.Label(brand, text=" 2.0", font=("Segoe UI", 14, "bold"), bg=BG2, fg=ACCENT).pack(side="left")
        tk.Label(hdr, text=f"{VERSION}  ·  Installer", font=F_SMALL, bg=BG2, fg=MUTED).pack(side="right", padx=22)
        tk.Frame(self, bg=BORDER, height=1).pack(fill="x")

        # Body container
        self.body = tk.Frame(self, bg=BG)
        self.body.pack(fill="both", expand=True)

        # Footer
        tk.Frame(self, bg=BORDER, height=1).pack(fill="x", side="bottom")
        ftr = tk.Frame(self, bg=BG2, height=54)
        ftr.pack(fill="x", side="bottom")
        ftr.pack_propagate(False)

        self._btn_back = tk.Button(
            ftr, text="← Home", font=F_BTN,
            bg=BG3, fg=MUTED, activebackground=BG3, activeforeground=TEXT,
            relief="flat", bd=0, padx=20, pady=7,
            cursor="hand2", command=self._show_welcome,
        )
        self._btn_back.pack(side="left", padx=16, pady=10)

        self._footer_lbl = tk.Label(ftr, text="", font=F_SMALL, bg=BG2, fg=MUTED)
        self._footer_lbl.pack(side="left", padx=8)

        self._btn_action = tk.Button(
            ftr, text="", font=F_BTN,
            bg=ACCENT, fg="white",
            activebackground=ACCENT_H, activeforeground="white",
            relief="flat", bd=0, padx=24, pady=7, cursor="hand2",
        )
        self._btn_action.pack(side="right", padx=16, pady=10)

    def _clear(self):
        for w in self.body.winfo_children():
            w.destroy()

    def _set_footer(self, *, back=False, action_text="", action_cmd=None, action_bg=ACCENT, status=""):
        if back:
            self._btn_back.config(state="normal", fg=MUTED, bg=BG3,
                                  activebackground=BG3, activeforeground=TEXT)
        else:
            self._btn_back.config(state="disabled", fg=BG2, bg=BG2,
                                  activebackground=BG2, activeforeground=BG2)

        if action_text and action_cmd:
            self._btn_action.config(
                state="normal", text=action_text, bg=action_bg,
                activebackground=ACCENT_H if action_bg == ACCENT else "#16a34a",
                fg="white", activeforeground="white", command=action_cmd,
            )
        else:
            self._btn_action.config(state="disabled", text="", bg=BG2,
                                    fg=BG2, activebackground=BG2, activeforeground=BG2)
        self._footer_lbl.config(text=status)

    # ── Welcome ─────────────────────────────────────────────────────────────────

    def _show_welcome(self):
        if self._poll_id:
            self.after_cancel(self._poll_id)
            self._poll_id = None
        self._clear()
        self._set_footer()

        frame = tk.Frame(self.body, bg=BG)
        frame.pack(fill="both", expand=True)
        tk.Frame(frame, bg=BG, height=34).pack()

        tk.Label(frame, text="⬡", font=("Segoe UI", 42), bg=BG, fg=ACCENT).pack()
        tk.Label(frame, text="ArcVault 2.0", font=("Segoe UI", 28, "bold"), bg=BG, fg=TEXT).pack(pady=(4, 2))
        tk.Label(frame, text=f"Automated Installer  ·  {VERSION}", font=F_SUB, bg=BG, fg=MUTED).pack()

        # Admin status badge
        if is_admin():
            badge_txt = "  ✓  Running as Administrator  "
            badge_bg, badge_fg = "#0d1f0d", SUCCESS
        else:
            badge_txt = "  ⚠  Not running as Administrator — click Coordinator or Agent to elevate  "
            badge_bg, badge_fg = WARN_BG, WARN_FG
        tk.Label(frame, text=badge_txt, font=F_SMALL,
                 bg=badge_bg, fg=badge_fg, pady=5).pack(pady=(14, 0))

        tk.Frame(frame, bg=BORDER, height=1, width=480).pack(pady=22)
        tk.Label(frame, text="What would you like to install?",
                 font=("Segoe UI", 12), bg=BG, fg=TEXT).pack(pady=(0, 18))

        row = tk.Frame(frame, bg=BG)
        row.pack()
        self._choice_card(row, "◈", "Coordinator",
                          "Central hub · manages agents\nstores jobs · hosts dashboard",
                          self._start_coordinator).pack(side="left", padx=12)
        self._choice_card(row, "◇", "Agent",
                          "Installs on client machines\nconnects back to coordinator",
                          self._show_agent_config).pack(side="left", padx=12)

        tk.Label(frame,
                 text="Install the Coordinator first, then run this installer on each machine you want to back up.",
                 font=F_SMALL, bg=BG, fg=MUTED, wraplength=620).pack(pady=(24, 0))

    def _choice_card(self, parent, icon, title, desc, on_click):
        card = tk.Frame(parent, bg=BG3, padx=28, pady=26,
                        highlightthickness=1, highlightbackground=BORDER)
        tk.Label(card, text=icon, font=("Segoe UI", 30), bg=BG3, fg=ACCENT).pack()
        tk.Label(card, text=title, font=("Segoe UI", 13, "bold"), bg=BG3, fg=TEXT).pack(pady=(8, 4))
        tk.Label(card, text=desc, font=F_SMALL, bg=BG3, fg=MUTED, justify="center").pack()
        tk.Button(card, text=f"Install {title}  →", font=F_BTN,
                  bg=ACCENT, fg="white", activebackground=ACCENT_H, activeforeground="white",
                  relief="flat", bd=0, padx=14, pady=7,
                  cursor="hand2", command=on_click).pack(pady=(18, 0))
        card.bind("<Enter>", lambda _: card.config(highlightbackground=ACCENT))
        card.bind("<Leave>", lambda _: card.config(highlightbackground=BORDER))
        return card

    # ── Agent config form ──────────────────────────────────────────────────────

    def _show_agent_config(self):
        self._clear()
        self._set_footer(back=True)

        outer = tk.Frame(self.body, bg=BG)
        outer.pack(fill="both", expand=True, padx=60, pady=30)

        tk.Label(outer, text="Agent Configuration", font=F_TITLE, bg=BG, fg=TEXT, anchor="w").pack(fill="x")
        tk.Label(outer, text="Enter the details below — the installer will handle everything else.",
                 font=F_SUB, bg=BG, fg=MUTED, anchor="w").pack(fill="x", pady=(4, 24))

        def entry_field(label_text, placeholder):
            tk.Label(outer, text=label_text, font=F_SMALL, bg=BG, fg=MUTED, anchor="w").pack(fill="x", pady=(0, 3))
            e = tk.Entry(outer, font=F_BODY, bg=BG3, fg=TEXT, insertbackground=TEXT,
                         relief="flat", bd=0, highlightthickness=1,
                         highlightbackground=BORDER, highlightcolor=ACCENT)
            e.insert(0, placeholder)
            e.bind("<FocusIn>", lambda ev, en=e, ph=placeholder: en.delete(0, "end") if en.get() == ph else None)
            e.pack(fill="x", ipady=8, padx=2, pady=(0, 14))
            return e

        e_url   = entry_field("Coordinator URL", "http://192.168.1.100:8080")
        e_name  = entry_field("Agent Name  (e.g. workstation-01, server-prod)", "agent-01")
        e_token = entry_field("Auth Token  (from: coordinator create-agent-token <name>)", "paste token here...")

        tk.Label(outer,
                 text="Run  coordinator create-agent-token <name>  on your coordinator machine to generate the token.",
                 font=F_SMALL, bg=BG, fg=MUTED, wraplength=720, justify="left").pack(fill="x")

        def validate_and_install():
            url   = e_url.get().strip()
            name  = e_name.get().strip()
            token = e_token.get().strip()
            if not url or url == "http://192.168.1.100:8080":
                messagebox.showwarning("Missing Info", "Please enter the Coordinator URL.")
                return
            if not name or name == "agent-01":
                messagebox.showwarning("Missing Info", "Please enter an Agent Name.")
                return
            if not token or len(token) < 32:
                messagebox.showwarning("Missing Info", "Please paste the Auth Token generated by 'coordinator create-agent-token'.")
                return
            self._run_install("agent", {"coordinator_url": url, "agent_name": name, "auth_token": token})

        self._set_footer(
            back=True,
            action_text="Install Agent  →",
            action_cmd=validate_and_install,
        )

    # ── Install screen ─────────────────────────────────────────────────────────

    def _start_coordinator(self):
        self._run_install("coordinator")

    def _run_install(self, component, agent_cfg=None):
        self._clear()
        self._set_footer()   # No back/action during install
        self.component = component
        comp_label = "Coordinator" if component == "coordinator" else "Agent"

        # ── Left: step list + progress ──────────────────────────────────────────
        left = tk.Frame(self.body, bg=BG3, width=230)
        left.pack(side="left", fill="y")
        left.pack_propagate(False)

        tk.Label(left, text=f"Installing {comp_label}",
                 font=("Segoe UI", 11, "bold"), bg=BG3, fg=TEXT,
                 padx=14, pady=14, anchor="w").pack(fill="x")
        tk.Frame(left, bg=BORDER, height=1).pack(fill="x")

        self._step_list = StepList(left)
        self._step_list.pack(fill="x", pady=6)

        prog_frame = tk.Frame(left, bg=BG3, padx=14, pady=10)
        prog_frame.pack(fill="x", side="bottom", pady=(0, 4))
        self._prog_lbl = tk.Label(prog_frame, text="Starting…", font=F_SMALL, bg=BG3, fg=MUTED)
        self._prog_lbl.pack(fill="x")
        self._prog_canvas = tk.Canvas(prog_frame, bg=BORDER, height=3,
                                      highlightthickness=0)
        self._prog_canvas.pack(fill="x", pady=(6, 0))

        # ── Divider ─────────────────────────────────────────────────────────────
        tk.Frame(self.body, bg=BORDER, width=1).pack(side="left", fill="y")

        # ── Right: live log ─────────────────────────────────────────────────────
        right = tk.Frame(self.body, bg=BG)
        right.pack(side="left", fill="both", expand=True)
        tk.Label(right, text="Install Log", font=F_SMALL, bg=BG, fg=MUTED,
                 padx=16, pady=10, anchor="w").pack(fill="x")
        tk.Frame(right, bg=BORDER, height=1).pack(fill="x")
        self._log = LogView(right)
        self._log.pack(fill="both", expand=True)

        # ── Start runner ────────────────────────────────────────────────────────
        self._log.append(f"Starting {comp_label} installation…", style="info")
        self.runner = InstallRunner(component, agent_cfg)
        self.runner.start()
        self._poll()

    def _poll(self):
        try:
            while True:
                self._dispatch(self.runner.q.get_nowait())
        except queue.Empty:
            pass
        self._poll_id = self.after(40, self._poll)

    def _dispatch(self, msg):
        ev = msg["event"]

        if ev == "steps":
            self._step_list.load(msg["steps"])
            self._total = len(msg["steps"])
            self._current = -1

        elif ev == "step_start":
            self._current = msg["index"]
            self._step_list.set_status(msg["index"], "running")
            pct = self._current / self._total * 100
            self._draw_bar(pct)
            self._prog_lbl.config(text=f"Step {msg['index'] + 1} of {self._total}")
            self._log.append(f"\n▶  {msg['name']}", style="info")

        elif ev == "step_done":
            self._step_list.set_status(msg["index"], "done")
            pct = (msg["index"] + 1) / self._total * 100
            self._draw_bar(pct)

        elif ev == "step_fail":
            self._step_list.set_status(msg["index"], "fail")
            self._log.append(f"\n✗  {msg['message']}", style="error")
            self._prog_lbl.config(text="Installation failed")
            self._set_footer(back=True, status="Installation failed")

        elif ev == "log":
            self._log.append(msg["text"], style=msg.get("style", "normal"))

        elif ev == "progress":
            self._prog_lbl.config(text=f"Downloading… {msg['pct']:.0f}%")
            self._draw_bar(msg["pct"])

        elif ev == "token":
            self.admin_token = msg["token"]

        elif ev == "done":
            self._draw_bar(100)
            self._prog_lbl.config(text="Complete")
            self.after(500, self._show_complete)

        elif ev == "fatal":
            self._log.append(f"Fatal: {msg['message']}", style="error")
            self._set_footer(back=True, status="Fatal error — see log")

    def _draw_bar(self, pct):
        c = self._prog_canvas
        c.update_idletasks()
        w = c.winfo_width()
        c.delete("all")
        c.create_rectangle(0, 0, w, 3, fill=BORDER, outline="")
        c.create_rectangle(0, 0, int(w * pct / 100), 3, fill=ACCENT, outline="")

    # ── Complete screen ────────────────────────────────────────────────────────

    def _show_complete(self):
        if self._poll_id:
            self.after_cancel(self._poll_id)
            self._poll_id = None
        self._clear()
        self._set_footer(back=True)

        frame = tk.Frame(self.body, bg=BG)
        frame.pack(fill="both", expand=True)
        tk.Frame(frame, bg=BG, height=46).pack()

        comp = "Coordinator" if self.component == "coordinator" else "Agent"
        tk.Label(frame, text="✓", font=("Segoe UI", 52), bg=BG, fg=SUCCESS).pack()
        tk.Label(frame, text=f"{comp} Installed Successfully",
                 font=("Segoe UI", 20, "bold"), bg=BG, fg=TEXT).pack(pady=(8, 4))

        if self.component == "coordinator":
            tk.Label(frame, text=f"Installed to {COORD_DIR}  ·  Service: {COORD_SVC}",
                     font=F_SMALL, bg=BG, fg=MUTED).pack()

            # Admin token display
            if self.admin_token:
                tk.Label(frame, text="Admin token — save this now:",
                         font=F_SMALL, bg=BG, fg=MUTED).pack(pady=(18, 5))
                token_frame = tk.Frame(frame, bg=BG3,
                                       highlightthickness=1, highlightbackground=ACCENT,
                                       padx=20, pady=12)
                token_frame.pack()
                tk.Label(token_frame, text=self.admin_token, font=F_TOKEN,
                         bg=BG3, fg=SUCCESS).pack(side="left")
                copy_lbl = tk.Label(token_frame, text="  Copy  ", font=F_SMALL,
                                    bg="#1c2135", fg=MUTED, padx=8, pady=3, cursor="hand2")
                copy_lbl.pack(side="left", padx=(14, 0))

                def copy_token(_=None):
                    self.clipboard_clear()
                    self.clipboard_append(self.admin_token)
                    copy_lbl.config(text="  ✓ Copied!  ", fg=SUCCESS)
                    self.after(1800, lambda: copy_lbl.config(text="  Copy  ", fg=MUTED))

                copy_lbl.bind("<Button-1>", copy_token)
                tk.Label(frame, text="Store in a password manager. This token will not be shown again.",
                         font=F_SMALL, bg=BG, fg=WARN_FG).pack(pady=(6, 0))

            tk.Frame(frame, bg=BORDER, height=1, width=440).pack(pady=22)

            tk.Button(frame, text="Open Dashboard  ↗", font=F_BTN,
                      bg=ACCENT, fg="white", activebackground=ACCENT_H, activeforeground="white",
                      relief="flat", bd=0, padx=20, pady=9, cursor="hand2",
                      command=lambda: webbrowser.open(DASHBOARD)).pack(pady=(0, 10))

            tk.Label(frame, text="Next: install agents on each machine you want to back up.",
                     font=F_SMALL, bg=BG, fg=MUTED).pack(pady=(6, 8))
            tk.Button(frame, text="Install an Agent  →", font=F_BTN,
                      bg=BG3, fg=TEXT, activebackground=BORDER, activeforeground=TEXT,
                      relief="flat", bd=0, padx=20, pady=9, cursor="hand2",
                      command=self._show_agent_config).pack()

        else:
            tk.Label(frame, text=f"Installed to {AGENT_DIR}  ·  Service: {AGENT_SVC}",
                     font=F_SMALL, bg=BG, fg=MUTED).pack()
            tk.Frame(frame, bg=BORDER, height=1, width=440).pack(pady=22)
            tk.Button(frame, text="View in Dashboard  ↗", font=F_BTN,
                      bg=ACCENT, fg="white", activebackground=ACCENT_H, activeforeground="white",
                      relief="flat", bd=0, padx=20, pady=9, cursor="hand2",
                      command=lambda: webbrowser.open(DASHBOARD)).pack()

        home = tk.Label(frame, text="← Install another component",
                        font=F_SMALL, bg=BG, fg=MUTED, cursor="hand2")
        home.pack(pady=(20, 0))
        home.bind("<Button-1>", lambda _: self._show_welcome())
        home.bind("<Enter>", lambda e: home.config(fg=TEXT))
        home.bind("<Leave>", lambda e: home.config(fg=MUTED))


# ── Entry point ────────────────────────────────────────────────────────────────

if __name__ == "__main__":
    # Request UAC elevation before the window opens if not already admin.
    # On Windows, this triggers the familiar "Do you want to allow this app
    # to make changes?" prompt. The app then re-launches elevated.
    if not is_admin():
        elevate_and_restart()

    app = ArcVaultInstaller()
    app.mainloop()
