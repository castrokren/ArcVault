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


# ── Design tokens — mirrors coordinator dashboard (style.css) ──────────────────
_BG_BASE      = "#07090e"
_BG_SURFACE   = "#0c0f18"
_BG_CARD      = "#11141f"
_BG_ELEVATED  = "#171b29"
_BG_INPUT     = "#0a0c14"

_BORDER_SUBTLE  = "#191d2c"
_BORDER_DEFAULT = "#222638"
_BORDER_STRONG  = "#2e3450"

_TEXT_PRIMARY   = "#dde4f2"
_TEXT_SECONDARY = "#7b87a2"
_TEXT_MUTED     = "#404b62"

_ACCENT       = "#00d4aa"
_ACCENT_HOVER = "#00b894"

_COLOR_WARNING = "#f59e0b"
_COLOR_ERROR   = "#ff4d6d"

# ── ArcVault icon — favicon.svg rendered to 64x64 PNG, base64 ─────────────────
_ICON_B64 = (
    "iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAABmJLR0QA/wD/"
    "AP+gvaeTAAAECElEQVR4nOXbXYimYxgH8N87H+bD52BbrI0lByyRtOVAlJJI"
    "CSFxhPJRElFKTlCb5GAjSdrkIznyVeQA68ARyVdrhV1mFwe7a2fM2NmZ2Xcc"
    "XKYd47Hv875zP/Pcr/3X/+x97uf53/d1X9d1X9f9coij0ebve3Es7sO16McE"
    "vsVc2k9rC304++/vgV/xKN5N/aKjcBemheCc+V4ZQT1tiB/EGjGz/S1+mwPG"
    "Ug94Dj5S/8qW4c9Ym1L8CXgczQzEteI0bsZwKvH9uB7jGYgrww04MZX4Ppzi"
    "gJfPnV9jRSrxRMh7PQNhZfgHzpfQQa/E7ZjJQFwrNvEwBrSf2xRiEJfitwzE"
    "leHbWJ1C+DxOx/sZCCvD7cL0k6EXT+mOkDeDKzCUSvzhuE44lLrFleGzOEbC"
    "fb8W32UgrAy/EPG+N4X4Bk4VM1q3sDIcw2UphM9jCHdgNgNxrdjEg8Jik+Fy"
    "cYauW1wZviGy02RYg00ZCCvDaf8sfCwZPbgNF6UasGJMYpcIf8kwpv6VLcv9"
    "OE8iz094/+1YlWrAZcA3eACbhUV0giamMNnATbgBR0uTUAzhLJFUVYVxcUTv"
    "dALmsA2PpfqghejB1bojqmxItpcWYA47hKe+pILxU+LjKge/Uf0rfDBuwep2"
    "yuLt4DicW9HYKbBHNHdGqxi8RxRTflH/KhdxVqT9fVWJP17srbqF/hefkLhw"
    "uhAjWC8SlrqFFnETTlbR6o+IQ9VEBkKLOCryk0raer1YJ+rydQst4oToZleG"
    "VXg+A6FFbOIhCVtli3EY7sW+DMQWcaPEJfOFGMSF+DIDoUX8SlhnFRmvXjGz"
    "L2YgtIg7caaKPD6chPuxNwOxizklWuSViR8WXvX7DMQWcb1IyCrDOnySgdAi"
    "vinqnJWgR1yUekmerbNtYmsm6RQV4Qjcid8zELuYY7hAhOXKsBJbMxC7mDO4"
    "Wwf3A9r1knMqiqlLxHPiJsu+dh9sd68M4xrcKnxBWfwpTHQndgtLugpHtvn+"
    "InwgbrL82MnD7U5AQ1R9B3QeY+fN9gy8gtM6HIeoPV6Mn0Sho2vQEFnaRp3v"
    "+0lxOSJpk3Q5sUKYbifi9+MRCTK9qoqiZTArfEMneBlPS2D2dU5Ap/hUdHT2"
    "pBis2yZgl0jEdohMdMmo7LRUAaZxj2iO7k01aLdMwByewaupB+6WLfCWOOL+"
    "rzCCK7UOeZvFnzUqScHrtIBJrVPhcZF2/yBif3LU6QMGHHxVZ0W1+XMJnd5i"
    "1GkBrSb/Bbwm6nu1fURd+FA0MStb+XnkGAVGxU3QrcvxsjonYM6/s7kp3OJA"
    "j7Fy1H0YWpjPN8X1t890UNnpRvSJCu47wuyfFH/SWtZFqax8XBJ9onfXEDF/"
    "d72fcwjiL+el9OvaTJIIAAAAAElFTkSuQmCC"
)


class ArcVaultInstaller:
    """Main installer class"""

    COORD_DIR = Path("C:/ArcVault")
    AGENT_DIR = Path("C:/ArcVault-Agent")

    def __init__(self):
        self.version = "0.4.0"
        self.components = set()
        self.coordinator_port = 8080
        self.admin_token = ""
        self.agent_id = ""
        self.agent_token = ""
        self.coordinator_url = ""
        self.credential_key = ""
        self.root = None

    def _setup_styles(self):
        style = ttk.Style()
        try:
            style.theme_use("clam")
        except Exception:
            pass

        style.configure("TFrame", background=_BG_BASE)
        style.configure("TLabel",
                        background=_BG_BASE,
                        foreground=_TEXT_SECONDARY,
                        font=("Segoe UI", 10))

        style.configure("TCheckbutton",
                        background=_BG_CARD,
                        foreground=_TEXT_PRIMARY,
                        font=("Segoe UI", 10),
                        indicatorcolor=_BG_ELEVATED,
                        focusthickness=0,
                        focuscolor=_BG_CARD)
        style.map("TCheckbutton",
                  background=[("active", _BG_CARD)],
                  foreground=[("active", _TEXT_PRIMARY)],
                  indicatorcolor=[("selected", _ACCENT)])

        style.configure("TEntry",
                        fieldbackground=_BG_INPUT,
                        background=_BG_INPUT,
                        foreground=_TEXT_PRIMARY,
                        insertcolor=_ACCENT,
                        borderwidth=1,
                        relief="flat",
                        font=("Segoe UI", 10),
                        padding=(6, 5))
        style.map("TEntry",
                  fieldbackground=[("readonly", _BG_ELEVATED), ("disabled", _BG_ELEVATED)],
                  foreground=[("readonly", _TEXT_MUTED), ("disabled", _TEXT_MUTED)])

        style.configure("Primary.TButton",
                        background=_ACCENT,
                        foreground=_BG_BASE,
                        font=("Segoe UI", 10, "bold"),
                        padding=(20, 8),
                        borderwidth=0,
                        focusthickness=0,
                        relief="flat")
        style.map("Primary.TButton",
                  background=[("active", _ACCENT_HOVER), ("pressed", _ACCENT_HOVER)],
                  foreground=[("active", _BG_BASE)])

        style.configure("Secondary.TButton",
                        background=_BG_ELEVATED,
                        foreground=_TEXT_SECONDARY,
                        font=("Segoe UI", 10),
                        padding=(20, 8),
                        borderwidth=1,
                        focusthickness=0,
                        relief="flat")
        style.map("Secondary.TButton",
                  background=[("active", _BG_CARD), ("pressed", _BG_CARD)],
                  foreground=[("active", _TEXT_PRIMARY)])

        style.configure("TProgressbar",
                        troughcolor=_BG_ELEVATED,
                        background=_ACCENT,
                        borderwidth=0,
                        thickness=3)

    def run(self):
        self.root = tk.Tk()
        self.root.title(f"ArcVault {self.version} Setup")
        self.root.geometry("640x460")
        self.root.resizable(False, False)
        self.root.configure(bg=_BG_BASE)
        self._setup_styles()

        try:
            self._icon = tk.PhotoImage(data=_ICON_B64)
            self.root.iconphoto(True, self._icon)
        except Exception:
            pass

        self.show_component_selection()
        self.root.mainloop()

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _header(self, parent, title, subtitle=None):
        strip = tk.Frame(parent, bg=_BG_SURFACE, height=62)
        strip.pack(fill=tk.X, side=tk.TOP)
        strip.pack_propagate(False)

        tk.Frame(strip, bg=_ACCENT, height=3).pack(fill=tk.X, side=tk.TOP)

        inner = tk.Frame(strip, bg=_BG_SURFACE)
        inner.pack(fill=tk.BOTH, expand=True, padx=24)

        text_col = tk.Frame(inner, bg=_BG_SURFACE)
        text_col.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, pady=10)

        tk.Label(text_col, text=title,
                 bg=_BG_SURFACE, fg=_TEXT_PRIMARY,
                 font=("Segoe UI", 13, "bold"), anchor="w").pack(anchor="w")
        if subtitle:
            tk.Label(text_col, text=subtitle,
                     bg=_BG_SURFACE, fg=_TEXT_MUTED,
                     font=("Segoe UI", 9), anchor="w").pack(anchor="w")

        tk.Label(inner, text=f"v{self.version}",
                 bg=_BG_ELEVATED, fg=_TEXT_MUTED,
                 font=("Consolas", 8),
                 padx=7, pady=3).pack(side=tk.RIGHT, anchor="center")

    def _nav_buttons(self, parent, next_text, next_cmd, back_cmd=None):
        tk.Frame(parent, bg=_BORDER_SUBTLE, height=1).pack(fill=tk.X, side=tk.BOTTOM)

        bf = tk.Frame(parent, bg=_BG_SURFACE, pady=14, padx=20)
        bf.pack(fill=tk.X, side=tk.BOTTOM)

        ttk.Button(bf, text=next_text, style="Primary.TButton",
                   command=next_cmd).pack(side=tk.RIGHT)
        if back_cmd:
            ttk.Button(bf, text="Back", style="Secondary.TButton",
                       command=back_cmd).pack(side=tk.LEFT)
        else:
            ttk.Button(bf, text="Cancel", style="Secondary.TButton",
                       command=self.root.quit).pack(side=tk.LEFT)

    # ------------------------------------------------------------------
    # Screens
    # ------------------------------------------------------------------

    def show_component_selection(self):
        self.clear_window()
        self.root.configure(bg=_BG_BASE)

        coordinator_var = tk.BooleanVar()
        agent_var = tk.BooleanVar()

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

        self._header(self.root, "ArcVault Setup Wizard",
                     "Backup orchestration for self-hosted infrastructure")
        self._nav_buttons(self.root, "Next", next_step)

        frame = tk.Frame(self.root, bg=_BG_BASE)
        frame.pack(fill=tk.BOTH, expand=True, padx=28, pady=20)

        tk.Label(frame, text="Welcome to ArcVault",
                 bg=_BG_BASE, fg=_TEXT_PRIMARY,
                 font=("Segoe UI", 20, "bold")).pack(anchor="w", pady=(0, 4))
        tk.Label(frame,
                 text=f"Version {self.version}  ·  Select the components to install",
                 bg=_BG_BASE, fg=_TEXT_MUTED,
                 font=("Segoe UI", 9)).pack(anchor="w", pady=(0, 18))

        cards_row = tk.Frame(frame, bg=_BG_BASE)
        cards_row.pack(fill=tk.X)

        _BUBBLE = 28

        def draw_bubble(canvas, selected):
            canvas.delete("all")
            r = _BUBBLE // 2 - 2
            cx = cy = _BUBBLE // 2
            if selected:
                canvas.create_oval(cx - r, cy - r, cx + r, cy + r,
                                   fill=_ACCENT, outline=_ACCENT)
                canvas.create_line(cx - 5, cy, cx - 1, cy + 4,
                                   fill=_BG_BASE, width=2, capstyle="round")
                canvas.create_line(cx - 1, cy + 4, cx + 5, cy - 3,
                                   fill=_BG_BASE, width=2, capstyle="round")
            else:
                canvas.create_oval(cx - r, cy - r, cx + r, cy + r,
                                   fill="", outline=_BORDER_STRONG, width=2)

        for var, label, desc in [
            (coordinator_var, "Coordinator",
             "Central server — manages jobs,\nschedules, and agents"),
            (agent_var, "Agent",
             "Runs on each machine to\nexecute backup jobs"),
        ]:
            card = tk.Frame(cards_row, bg=_BG_CARD,
                            highlightbackground=_BORDER_DEFAULT, highlightthickness=2,
                            cursor="hand2")
            card.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=(0, 10))

            bubble = tk.Canvas(card, width=_BUBBLE, height=_BUBBLE,
                               bg=_BG_CARD, highlightthickness=0)
            bubble.pack(side=tk.LEFT, padx=(16, 12), pady=20)
            draw_bubble(bubble, False)

            col = tk.Frame(card, bg=_BG_CARD)
            col.pack(side=tk.LEFT, fill=tk.Y, pady=16, expand=True)

            title_lbl = tk.Label(col, text=label, bg=_BG_CARD, fg=_TEXT_PRIMARY,
                                 font=("Segoe UI", 10, "bold"))
            title_lbl.pack(anchor="w")
            desc_lbl = tk.Label(col, text=desc, bg=_BG_CARD, fg=_TEXT_MUTED,
                                font=("Segoe UI", 9), justify=tk.LEFT)
            desc_lbl.pack(anchor="w", pady=(3, 0))

            def make_toggle(v, c, bub):
                def toggle(event=None):
                    v.set(not v.get())
                    if v.get():
                        c.configure(highlightbackground=_ACCENT)
                        draw_bubble(bub, True)
                    else:
                        c.configure(highlightbackground=_BORDER_DEFAULT)
                        draw_bubble(bub, False)
                return toggle

            toggle = make_toggle(var, card, bubble)
            for w in (card, bubble, col, title_lbl, desc_lbl):
                w.bind("<Button-1>", toggle)

        tk.Label(frame,
                 text="Click a card to select it. You can install both on the same machine.",
                 bg=_BG_BASE, fg=_TEXT_MUTED,
                 font=("Segoe UI", 9)).pack(anchor="w", pady=(14, 0))

    def show_coordinator_config(self):
        self.clear_window()
        self.root.configure(bg=_BG_BASE)

        self._header(self.root, "Coordinator Configuration",
                     "Network and security settings for the server")

        frame = tk.Frame(self.root, bg=_BG_BASE)
        frame.pack(fill=tk.BOTH, expand=True)

        def row(lbl, **kw):
            r = tk.Frame(frame, bg=_BG_BASE, padx=28)
            r.pack(fill=tk.X, pady=4)
            tk.Label(r, text=lbl, bg=_BG_BASE, fg=_TEXT_SECONDARY,
                     font=("Segoe UI", 10), width=18, anchor=tk.W).pack(side=tk.LEFT)
            e = ttk.Entry(r, **kw)
            e.pack(side=tk.LEFT, padx=(8, 0), fill=tk.X, expand=True)
            return e

        tk.Frame(frame, bg=_BG_BASE, height=18).pack()
        port_var = row("Port:", width=10)
        port_var.insert(0, "8080")

        tk.Label(frame,
                 text="Default login: admin / changeme  (you'll be prompted to change it on first login)",
                 bg=_BG_BASE, fg=_TEXT_MUTED,
                 font=("Segoe UI", 9)).pack(anchor="w", padx=28, pady=(8, 0))

        card = tk.Frame(frame, bg=_BG_CARD,
                        highlightbackground=_BORDER_DEFAULT, highlightthickness=1)
        card.pack(fill=tk.X, padx=28, pady=(16, 0))
        tk.Label(card,
                 text="ℹ  Installs to C:\\ArcVault  ·  Registered as arcvault-coordinator Windows service",
                 bg=_BG_CARD, fg=_TEXT_MUTED,
                 font=("Segoe UI", 9)).pack(anchor="w", padx=14, pady=10)

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

        self._nav_buttons(frame, "Next", next_step,
                          self.show_component_selection)

    def show_agent_config(self):
        self.clear_window()
        self.root.configure(bg=_BG_BASE)

        self._header(self.root, "Agent Configuration",
                     "Connection details for this agent")

        frame = tk.Frame(self.root, bg=_BG_BASE)
        frame.pack(fill=tk.BOTH, expand=True)

        def row(lbl, **kw):
            r = tk.Frame(frame, bg=_BG_BASE, padx=28)
            r.pack(fill=tk.X, pady=4)
            tk.Label(r, text=lbl, bg=_BG_BASE, fg=_TEXT_SECONDARY,
                     font=("Segoe UI", 10), width=18, anchor=tk.W).pack(side=tk.LEFT)
            e = ttk.Entry(r, **kw)
            e.pack(side=tk.LEFT, padx=(8, 0), fill=tk.X, expand=True)
            return e

        tk.Frame(frame, bg=_BG_BASE, height=18).pack()

        if "coordinator" in self.components:
            default_url = f"https://localhost"
            self.coordinator_url = default_url
            url_var = row("Coordinator URL:", width=36)
            url_var.insert(0, default_url)
            url_var.config(state="readonly")
        else:
            url_var = row("Coordinator URL:", width=36)

        id_var = row("Agent ID:", width=24)
        id_var.insert(0, os.environ.get("COMPUTERNAME", "agent-1"))

        if "coordinator" in self.components:
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

        self._nav_buttons(frame, "Next", next_step,
                          self.show_component_selection)

    def show_review(self):
        self.clear_window()
        self.root.configure(bg=_BG_BASE)

        self._header(self.root, "Review Configuration",
                     "Confirm your settings before installing")

        frame = tk.Frame(self.root, bg=_BG_BASE)
        frame.pack(fill=tk.BOTH, expand=True)

        lines = [f"Components:  {', '.join(c.title() for c in sorted(self.components))}"]
        if "coordinator" in self.components:
            lines += [f"Port:  {self.coordinator_port}",
                      f"Install dir:  {self.COORD_DIR}",
                      f"Default login:  admin / changeme"]
        if "agent" in self.components:
            lines += [f"Coordinator URL:  {self.coordinator_url}",
                      f"Agent ID:  {self.agent_id}"]

        tk.Frame(frame, bg=_BG_BASE, height=18).pack()

        card = tk.Frame(frame, bg=_BG_CARD,
                        highlightbackground=_BORDER_DEFAULT, highlightthickness=1)
        card.pack(fill=tk.X, padx=28)
        card_inner = tk.Frame(card, bg=_BG_CARD, padx=18, pady=14)
        card_inner.pack(fill=tk.X)

        for line in lines:
            if ":  " in line:
                key, _, val = line.partition(":  ")
                row_f = tk.Frame(card_inner, bg=_BG_CARD)
                row_f.pack(fill=tk.X, pady=3)
                tk.Label(row_f, text=key.strip() + ":", bg=_BG_CARD,
                         fg=_TEXT_MUTED, font=("Segoe UI", 9),
                         width=16, anchor="w").pack(side=tk.LEFT)
                tk.Label(row_f, text=val.strip(), bg=_BG_CARD,
                         fg=_TEXT_PRIMARY, font=("Segoe UI", 10)).pack(side=tk.LEFT)
            else:
                tk.Label(card_inner, text=line, bg=_BG_CARD,
                         fg=_TEXT_PRIMARY, font=("Segoe UI", 10)).pack(anchor="w", pady=3)

        tk.Label(frame, text="Click Install to begin.",
                 bg=_BG_BASE, fg=_TEXT_MUTED,
                 font=("Segoe UI", 9)).pack(anchor="w", padx=28, pady=(12, 0))

        self._nav_buttons(frame, "Install", self.install,
                          self.show_agent_config if "agent" in self.components
                          else self.show_coordinator_config)

    def install(self):
        self.clear_window()
        self.root.configure(bg=_BG_BASE)

        self._header(self.root, "Installing ArcVault...",
                     "Please wait while services are configured")

        frame = tk.Frame(self.root, bg=_BG_BASE)
        frame.pack(fill=tk.BOTH, expand=True, padx=28, pady=24)

        status = tk.Label(frame, text="Preparing...",
                          bg=_BG_BASE, fg=_TEXT_SECONDARY,
                          font=("Segoe UI", 10))
        status.pack(anchor="w", pady=(0, 10))

        progress = ttk.Progressbar(frame, length=560, mode="indeterminate",
                                   style="TProgressbar")
        progress.pack(fill=tk.X, pady=(0, 20))
        progress.start(10)

        steps_frame = tk.Frame(frame, bg=_BG_CARD,
                               highlightbackground=_BORDER_DEFAULT, highlightthickness=1)
        steps_frame.pack(fill=tk.X)
        steps_inner = tk.Frame(steps_frame, bg=_BG_CARD, padx=18, pady=14)
        steps_inner.pack(fill=tk.X)

        for step in ["Write configuration", "Install services",
                     "Start services", "Open dashboard"]:
            r = tk.Frame(steps_inner, bg=_BG_CARD)
            r.pack(anchor="w", pady=3)
            tk.Label(r, text="·", bg=_BG_CARD, fg=_BORDER_STRONG,
                     font=("Segoe UI", 14)).pack(side=tk.LEFT, padx=(0, 8))
            tk.Label(r, text=step, bg=_BG_CARD, fg=_TEXT_MUTED,
                     font=("Segoe UI", 9)).pack(side=tk.LEFT)

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
    # Functional helpers (unchanged)
    # ------------------------------------------------------------------

    def generate_credential_key(self):
        return secrets.token_hex(32)

    def get_or_create_credential_key(self):
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
        return self.generate_credential_key(), False

    def set_service_environment_variable(self, service_name, var_name, var_value):
        try:
            reg_path = f'HKLM\\SYSTEM\\CurrentControlSet\\Services\\{service_name}\\Environment'
            subprocess.run(
                ['reg', 'add', reg_path, '/v', var_name, '/d', var_value, '/f'],
                capture_output=True, text=True, shell=True, check=True
            )
        except Exception as e:
            print(f"Warning: Failed to set service environment variable: {e}")

    def write_coordinator_config(self):
        self.COORD_DIR.mkdir(parents=True, exist_ok=True)
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
        if not is_existing:
            messagebox.showinfo(
                "Credential Key Generated",
                f"A credential encryption key has been generated:\n\n{key}\n\n"
                "Save this key in a secure location!\n"
                "You will need it to re-install or migrate the system.\n\n"
                "The key is stored in C:\\ArcVault\\config.json."
            )

    def write_agent_config(self):
        self.AGENT_DIR.mkdir(parents=True, exist_ok=True)

        # Strip trailing slash from coordinator URL to avoid double-slash in API paths
        coordinator_url = self.coordinator_url.rstrip("/")

        # Copy coordinator cert for TLS pinning
        cert_dest = self.AGENT_DIR / "coordinator.crt"
        coord_cert = self.COORD_DIR / "cert.pem"
        if coord_cert.exists():
            shutil.copy2(str(coord_cert), str(cert_dest))

        content = (
            f"agent_id: {self.agent_id}\n"
            f"coordinator_url: {coordinator_url}\n"
            f"auth_token: {self.agent_token}\n"
            f"ca_cert_file: {cert_dest}\n"
        )
        with open(self.AGENT_DIR / "agent-config.yaml", "w") as f:
            f.write(content)

    def install_service(self, service_type):
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

            install_dir.mkdir(parents=True, exist_ok=True)

            subprocess.run(["sc", "stop", service_name],
                           capture_output=True, text=True, shell=True)
            time.sleep(1)
            subprocess.run(["sc", "delete", service_name],
                           capture_output=True, text=True, shell=True)

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
            time.sleep(2)

            binary_dst = install_dir / binary_name
            shutil.copy(binary_src, binary_dst)

            result = subprocess.run(
                [str(binary_dst), "install-service"],
                capture_output=True, text=True, shell=True,
            )
            if result.returncode != 0:
                raise Exception(f"install-service failed: {result.stdout} {result.stderr}")

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

    def open_browser(self, url):
        try:
            webbrowser.open(url)
        except Exception:
            pass

    def generate_token(self, length=32):
        return secrets.token_hex(length)

    def clear_window(self):
        for w in self.root.winfo_children():
            w.destroy()


if __name__ == "__main__":
    ArcVaultInstaller().run()
