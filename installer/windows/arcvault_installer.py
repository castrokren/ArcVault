#!/usr/bin/env python3
"""
ArcVault Windows Installer
Creates a native Windows installer for ArcVault (Coordinator, Agent, or Both)
Can be compiled to .exe using PyInstaller
"""

import os
import sys
import json
import re
import shutil
import subprocess
import winreg
import time
import secrets
import webbrowser
from pathlib import Path
import tkinter as tk
from tkinter import ttk, messagebox
import threading


# ── Design tokens — mirrors coordinator dashboard "Kiln" (src/style.css) ──────
# Warm charcoal + copper. Hex converted from the dashboard's OKLCH dark tokens.
_BG_BASE      = "#13100d"
_BG_SURFACE   = "#181411"
_BG_CARD      = "#1d1815"
_BG_ELEVATED  = "#27221d"
_BG_INPUT     = "#15120f"

_BORDER_SUBTLE  = "#28231f"
_BORDER_DEFAULT = "#352f2a"
_BORDER_STRONG  = "#4c433d"

_TEXT_PRIMARY   = "#ebe7e1"
_TEXT_SECONDARY = "#a0978b"
_TEXT_MUTED     = "#69625a"

_ACCENT       = "#e69b4c"  # copper
_ACCENT_HOVER = "#cf8634"

_COLOR_WARNING = "#e4b750"
_COLOR_ERROR   = "#ef6661"

# ── ArcVault icon — favicon.svg rendered to 64x64 PNG, base64 ─────────────────
_ICON_B64 = (
    "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"
)


class ArcVaultInstaller:
    """Main installer class"""

    COORD_DIR = Path("C:/ArcVault")
    AGENT_DIR = Path("C:/ArcVault-Agent")

    def __init__(self):
        self.version = "0.6.0"
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

        self.coordinator_port = 443

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
            # Minted per-agent during install, once coordinator.exe is in place.
            # Deliberately NOT the admin token: that is a permanent, fleet-wide
            # credential that cannot be revoked for one machine.
            token_var = row("Auth Token:", width=36)
            token_var.insert(0, "generated during install")
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

                status.config(text="Installing services...")
                self.root.update()

                # Order matters: the coordinator must be installed before the
                # agent, because minting the agent's own token runs
                # coordinator.exe against the config and DB it just created.
                if "coordinator" in self.components:
                    self.install_service("coordinator")

                if "agent" in self.components:
                    if "coordinator" in self.components:
                        status.config(text="Minting agent token...")
                        self.root.update()
                        self.agent_token = self.mint_agent_token(self.agent_id)
                    self.write_agent_config()
                    self.install_service("agent")

                if "coordinator" in self.components:
                    status.config(text="Waiting for coordinator to start...")
                    self.root.update()
                    time.sleep(3)
                    status.config(text="Opening dashboard...")
                    self.root.update()
                    self.open_browser(f"https://localhost:{self.coordinator_port}")

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
        """Return (key, is_existing).

        Recovering an existing key matters more than anything else here: a fresh
        key makes every stored `credential_profiles` row permanently
        undecryptable. Look in the service Environment first (its intended home),
        then config.json (the legacy location), and only then generate.
        """
        env = self._read_service_env("arcvault-coordinator")
        key = env.get("ARCVAULT_CREDENTIAL_KEY", "")
        if len(key) == 64:
            return key, True

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

    def get_or_create_jwt_secret(self, service_name="arcvault-coordinator"):
        # The coordinator clears jwt_secret from config.json and regenerates a
        # random one on every start unless ARCVAULT_JWT_SECRET is set — which
        # silently logs out every session on each service restart. Reuse any
        # existing value so an upgrade doesn't invalidate live sessions.
        existing = self._read_service_env(service_name).get("ARCVAULT_JWT_SECRET", "")
        if len(existing) == 64:
            return existing
        return secrets.token_hex(32)

    @staticmethod
    def parse_env_entries(items):
        """REG_MULTI_SZ strings -> {NAME: value}. Malformed entries are skipped.

        Values may themselves contain '=' (partition splits on the first only).
        """
        entries = {}
        for item in items:
            name, sep, value = item.partition("=")
            if sep and name:
                entries[name] = value
        return entries

    @staticmethod
    def format_env_entries(entries):
        """{NAME: value} -> sorted REG_MULTI_SZ strings."""
        return [f"{name}={value}" for name, value in sorted(entries.items())]

    def write_credential_key_to_config(self):
        """Last-resort fallback: put credential_key back into config.json.

        Only called when the registry write failed. `loadCredentialKey()` prefers
        cfg.CredentialKey over the env var, so this keeps encryption working at the
        cost of the key living next to the database.
        """
        config_path = self.COORD_DIR / "config.json"
        try:
            with open(config_path) as f:
                config = json.load(f)
            config["credential_key"] = self.credential_key
            with open(config_path, "w") as f:
                json.dump(config, f, indent=2)
        except Exception as exc:
            raise Exception(
                f"Could not store the credential key in the registry OR in "
                f"{config_path} ({exc}). The coordinator will not be able to "
                f"decrypt stored credentials."
            )

    def _read_service_env(self, service_name):
        """Read a service's environment as {NAME: value}.

        SCM injects exactly one thing: a REG_MULTI_SZ value named `Environment`,
        sitting directly ON the service key, holding `NAME=value` strings.

        Installer builds before 2026-07-24 ran
        `reg add ...\\Environment /v NAME /d VALUE`, which instead created a
        *subkey* named Environment containing REG_SZ values. SCM never reads that,
        so ARCVAULT_JWT_SECRET and ARCVAULT_CREDENTIAL_KEY were never actually
        injected — the coordinator logged "Generated new JWTSecret" on every start
        and read the credential key off disk. We still read the legacy subkey here
        so an upgrade can recover those secrets rather than minting new ones.
        """
        entries = {}
        path = rf"SYSTEM\CurrentControlSet\Services\{service_name}"

        try:
            with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, path) as key:
                raw, kind = winreg.QueryValueEx(key, "Environment")
                if kind == winreg.REG_MULTI_SZ:
                    entries.update(self.parse_env_entries(raw))
        except OSError:
            pass

        try:
            with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, path + r"\Environment") as key:
                index = 0
                while True:
                    try:
                        name, value, _ = winreg.EnumValue(key, index)
                    except OSError:
                        break
                    entries.setdefault(name, value)
                    index += 1
        except OSError:
            pass

        return entries

    def set_service_environment_variable(self, service_name, var_name, var_value):
        """Set NAME=value in the service key's REG_MULTI_SZ `Environment` value.

        Returns True only after reading the value back — a silent failure here is
        what left this machine with no injected environment at all. Merges with
        existing entries instead of replacing, so the credential key and the JWT
        secret coexist.
        """
        path = rf"SYSTEM\CurrentControlSet\Services\{service_name}"

        entries = self._read_service_env(service_name)
        entries[var_name] = var_value
        payload = self.format_env_entries(entries)

        try:
            with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, path, 0, winreg.KEY_SET_VALUE) as key:
                winreg.SetValueEx(key, "Environment", 0, winreg.REG_MULTI_SZ, payload)
        except OSError as exc:
            print(f"Warning: failed to write service Environment: {exc}")
            return False

        return self._read_service_env(service_name).get(var_name) == var_value

    def write_coordinator_config(self):
        self.COORD_DIR.mkdir(parents=True, exist_ok=True)
        key, is_existing = self.get_or_create_credential_key()
        self.credential_key = key

        # Preserve existing TLS cert paths if already configured
        existing_cert_file = ""
        existing_key_file = ""
        try:
            config_path = self.COORD_DIR / "config.json"
            if config_path.exists():
                with open(config_path) as f:
                    existing = json.load(f)
                existing_cert_file = existing.get("cert_file", "")
                existing_key_file = existing.get("key_file", "")
        except Exception:
            pass

        # No credential_key here on purpose. It belongs in the service Environment,
        # not in a file sitting in the same directory as arcvault.db — anyone who
        # can read the DB could otherwise read the key that protects it, making
        # encryption-at-rest decorative. install_service() writes it to the
        # registry and only falls back to this file if that write fails.
        config = {
            "port": self.coordinator_port,
            "admin_token": self.admin_token,
            "database_path": str(self.COORD_DIR / "arcvault.db"),
            "environment": "production",
            "allowed_origins": ["https://localhost"],
        }
        if existing_cert_file:
            config["cert_file"] = existing_cert_file
        if existing_key_file:
            config["key_file"] = existing_key_file

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

    def mint_agent_token(self, agent_id):
        """Mint a per-agent token via the coordinator CLI.

        Runs `coordinator.exe create-agent-token <id> --token-only`, which opens
        config.json and the DB sitting next to the exe -- no running service or
        network call required. Used instead of handing the agent the admin
        token: this one is a row in `tokens`, so it names one machine and can be
        revoked on its own.
        """
        exe = self.COORD_DIR / "coordinator.exe"
        if not exe.exists():
            raise Exception(f"coordinator.exe not found at {exe}")

        last_err = ""
        # The coordinator service is starting in parallel and may be migrating
        # the DB; a couple of retries covers the overlap.
        for attempt in range(3):
            result = subprocess.run(
                [str(exe), "create-agent-token", agent_id, "--token-only"],
                capture_output=True, text=True, shell=True,
            )
            token = result.stdout.strip()
            # CreateAgentToken returns 32 random bytes hex-encoded.
            if result.returncode == 0 and re.fullmatch(r"[0-9a-f]{64}", token):
                return token
            last_err = (result.stderr or result.stdout or "no output").strip()
            time.sleep(2)

        raise Exception(
            "Could not mint an agent token via coordinator.exe "
            f"(last error: {last_err}). The agent was not installed with a "
            "token; use 'Enroll Agent' in the dashboard to generate one."
        )

    def write_agent_config(self):
        self.AGENT_DIR.mkdir(parents=True, exist_ok=True)

        # Strip trailing slash from coordinator URL to avoid double-slash in API paths
        coordinator_url = self.coordinator_url.rstrip("/")

        # Copy the coordinator cert for TLS pinning. A ca_cert_file pointing at a
        # file that does not exist is worse than none at all: BuildTLSConfig
        # errors, and the agent then runs with its heartbeat disabled. Empty
        # means "use the system roots", so only write the key when we have a cert.
        cert_dest = self.AGENT_DIR / "coordinator.crt"
        coord_cert = self.COORD_DIR / "cert.pem"

        if "coordinator" in self.components:
            # Co-install: the coordinator service generates cert.pem on its first
            # start, which is racing us here. Its absence is a real failure.
            for _ in range(15):
                if coord_cert.exists():
                    break
                time.sleep(1)
            if not coord_cert.exists():
                raise Exception(
                    f"Coordinator certificate not found at {coord_cert} — the "
                    "coordinator service may have failed to start."
                )

        if coord_cert.exists():
            shutil.copy2(str(coord_cert), str(cert_dest))
            ca_line = f"ca_cert_file: {cert_dest}\n"
        else:
            # Agent-only install against a remote coordinator: we have no way to
            # fetch its cert here. System roots work for a CA-signed coordinator;
            # for the default self-signed one, "Enroll Agent" in the dashboard
            # generates a script that embeds the cert.
            ca_line = ""
            messagebox.showwarning(
                "No coordinator certificate",
                "This agent will verify the coordinator against the system trust "
                "store.\n\nIf the coordinator uses the default self-signed "
                "certificate, the agent will not be able to connect. Use "
                "'Enroll Agent' in the dashboard instead — it generates an "
                "install script with the certificate embedded."
            )

        content = (
            f"agent_id: {self.agent_id}\n"
            f"coordinator_url: {coordinator_url}\n"
            f"auth_token: {self.agent_token}\n"
            f"{ca_line}"
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
                # Must run AFTER install-service: `sc delete` + reinstall recreates
                # the service key, dropping whatever Environment value was there.
                cred_ok = self.set_service_environment_variable(
                    "arcvault-coordinator", "ARCVAULT_CREDENTIAL_KEY", self.credential_key)
                jwt_ok = self.set_service_environment_variable(
                    "arcvault-coordinator", "ARCVAULT_JWT_SECRET",
                    self.get_or_create_jwt_secret())

                if not cred_ok:
                    # Without the env var the coordinator cannot decrypt stored
                    # credentials at all (503 on credential endpoints). Fall back to
                    # the on-disk key: weaker, because it then sits beside the DB it
                    # protects, but functional. Never leave the operator with neither.
                    self.write_credential_key_to_config()
                    messagebox.showwarning(
                        "Credential key stored on disk",
                        "Could not write the credential key to the service registry, "
                        "so it was written to C:\\ArcVault\\config.json instead.\n\n"
                        "This works, but the key then sits in the same folder as the "
                        "database it protects. Re-run the installer as Administrator "
                        "to move it into the service environment."
                    )
                if not jwt_ok:
                    messagebox.showwarning(
                        "Sessions will not survive restarts",
                        "Could not write ARCVAULT_JWT_SECRET to the service registry.\n\n"
                        "The coordinator will generate a random JWT secret on every "
                        "start, so every dashboard session is invalidated whenever the "
                        "service restarts and logout-revocation stops being meaningful. "
                        "Re-run the installer as Administrator to fix this."
                    )

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


def _ensure_elevated():
    """Re-launch with UAC elevation if not already admin."""
    import ctypes
    if ctypes.windll.shell32.IsUserAnAdmin():
        return
    ctypes.windll.shell32.ShellExecuteW(
        None, "runas", sys.executable, " ".join(sys.argv), None, 1
    )
    sys.exit(0)


if __name__ == "__main__":
    _ensure_elevated()
    ArcVaultInstaller().run()
