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
import tempfile
from pathlib import Path
from typing import Optional, Dict
import tkinter as tk
from tkinter import ttk, messagebox, simpledialog
import threading
import socket


class ArcVaultInstaller:
    """Main installer class"""

    def __init__(self):
        self.version = "1.1.0"
        self.install_path = Path("C:/Program Files/ArcVault")
        self.config_path = Path.home() / ".arcvault"
        self.components = set()
        self.coordinator_port = 8080
        self.admin_username = ""
        self.admin_password = ""
        self.admin_token = ""
        self.agent_id = ""
        self.agent_token = ""
        self.coordinator_url = ""
        self.root = None
        self.current_step = 0

    def run(self):
        """Start the installer GUI"""
        self.root = tk.Tk()
        self.root.title(f"ArcVault {self.version} Setup Wizard")
        self.root.geometry("600x400")
        self.root.resizable(False, False)

        # Disable window resizing
        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(0, weight=1)

        self.show_welcome()
        self.root.mainloop()

    def show_welcome(self):
        """Welcome screen"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Welcome to ArcVault",
                         font=("Arial", 20, "bold"))
        title.pack(pady=20)

        subtitle = ttk.Label(frame, text=f"Version {self.version}",
                            font=("Arial", 12))
        subtitle.pack()

        description = ttk.Label(frame,
            text="User-friendly backup orchestration system\n\n"
                 "This wizard will guide you through installing ArcVault\n"
                 "on your Windows machine.",
            justify=tk.CENTER,
            font=("Arial", 10))
        description.pack(pady=20)

        info = ttk.Label(frame,
            text="You can choose to install:\n"
                 "• Coordinator (server)\n"
                 "• Agent (client)\n"
                 "• Both",
            justify=tk.LEFT,
            font=("Arial", 9))
        info.pack(pady=20)

        button_frame = ttk.Frame(frame)
        button_frame.pack(fill=tk.X, pady=20)

        ttk.Button(button_frame, text="Next >",
                  command=self.show_component_selection).pack(side=tk.RIGHT)
        ttk.Button(button_frame, text="Cancel",
                  command=self.root.quit).pack(side=tk.LEFT)

    def show_component_selection(self):
        """Component selection screen"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Select Components",
                         font=("Arial", 16, "bold"))
        title.pack(pady=20)

        # Coordinator checkbox
        coordinator_var = tk.BooleanVar()
        coordinator_check = ttk.Checkbutton(
            frame,
            text="Coordinator (Server) - Central backup orchestration",
            variable=coordinator_var,
            onvalue=True,
            offvalue=False
        )
        coordinator_check.pack(anchor=tk.W, pady=10)

        # Agent checkbox
        agent_var = tk.BooleanVar()
        agent_check = ttk.Checkbutton(
            frame,
            text="Agent (Client) - Backup execution on this machine",
            variable=agent_var,
            onvalue=True,
            offvalue=False
        )
        agent_check.pack(anchor=tk.W, pady=10)

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

            if "coordinator" in self.components and "agent" in self.components:
                self.show_coordinator_config()
            elif "coordinator" in self.components:
                self.show_coordinator_config()
            else:
                self.show_agent_config()

        button_frame = ttk.Frame(frame)
        button_frame.pack(fill=tk.X, pady=20)

        ttk.Button(button_frame, text="Next >",
                  command=next_step).pack(side=tk.RIGHT)
        ttk.Button(button_frame, text="< Back",
                  command=self.show_welcome).pack(side=tk.LEFT)

    def show_coordinator_config(self):
        """Coordinator configuration screen"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Coordinator Configuration",
                         font=("Arial", 16, "bold"))
        title.pack(pady=20)

        # Port
        port_frame = ttk.Frame(frame)
        port_frame.pack(fill=tk.X, pady=10)
        ttk.Label(port_frame, text="Port (default 8080):").pack(side=tk.LEFT)
        port_var = ttk.Entry(port_frame, width=15)
        port_var.insert(0, "8080")
        port_var.pack(side=tk.LEFT, padx=10)

        # Admin username
        user_frame = ttk.Frame(frame)
        user_frame.pack(fill=tk.X, pady=10)
        ttk.Label(user_frame, text="Admin Username:").pack(side=tk.LEFT)
        user_var = ttk.Entry(user_frame, width=30)
        user_var.pack(side=tk.LEFT, padx=10)

        # Admin password
        pass_frame = ttk.Frame(frame)
        pass_frame.pack(fill=tk.X, pady=10)
        ttk.Label(pass_frame, text="Admin Password:").pack(side=tk.LEFT)
        pass_var = ttk.Entry(pass_frame, width=30, show="*")
        pass_var.pack(side=tk.LEFT, padx=10)

        # HTTPS checkbox
        https_var = tk.BooleanVar()
        https_check = ttk.Checkbutton(frame, text="Enable HTTPS",
                                      variable=https_var)
        https_check.pack(anchor=tk.W, pady=10)

        def next_step():
            if not user_var.get():
                messagebox.showerror("Required", "Admin username required")
                return
            if not pass_var.get():
                messagebox.showerror("Required", "Admin password required")
                return

            try:
                self.coordinator_port = int(port_var.get())
            except ValueError:
                messagebox.showerror("Invalid", "Port must be a number")
                return

            self.admin_username = user_var.get()
            self.admin_password = pass_var.get()

            # Generate token
            self.admin_token = self.generate_token(32)

            if "agent" in self.components:
                self.show_agent_config()
            else:
                self.show_review()

        button_frame = ttk.Frame(frame)
        button_frame.pack(fill=tk.X, pady=20)

        ttk.Button(button_frame, text="Next >",
                  command=next_step).pack(side=tk.RIGHT)
        ttk.Button(button_frame, text="< Back",
                  command=self.show_component_selection).pack(side=tk.LEFT)

    def show_agent_config(self):
        """Agent configuration screen"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Agent Configuration",
                         font=("Arial", 16, "bold"))
        title.pack(pady=20)

        # Coordinator URL
        url_frame = ttk.Frame(frame)
        url_frame.pack(fill=tk.X, pady=10)
        ttk.Label(url_frame, text="Coordinator URL:").pack(side=tk.LEFT)

        # Pre-fill if coordinator is being installed
        if "coordinator" in self.components:
            default_url = f"http://localhost:{self.coordinator_port}"
            self.coordinator_url = default_url
            url_var = ttk.Entry(url_frame, width=40)
            url_var.insert(0, default_url)
            url_var.config(state="readonly")
        else:
            url_var = ttk.Entry(url_frame, width=40)
            url_var.pack(side=tk.LEFT, padx=10)

        # Agent ID
        id_frame = ttk.Frame(frame)
        id_frame.pack(fill=tk.X, pady=10)
        ttk.Label(id_frame, text="Agent ID:").pack(side=tk.LEFT)
        id_var = ttk.Entry(id_frame, width=30)
        id_var.insert(0, os.environ.get("COMPUTERNAME", "agent-1"))
        id_var.pack(side=tk.LEFT, padx=10)

        # Agent Token
        token_frame = ttk.Frame(frame)
        token_frame.pack(fill=tk.X, pady=10)
        ttk.Label(token_frame, text="Auth Token:").pack(side=tk.LEFT)

        if "coordinator" in self.components:
            self.agent_token = self.generate_token(32)
            token_var = ttk.Entry(token_frame, width=40)
            token_var.insert(0, self.agent_token[:16] + "...")
            token_var.config(state="readonly")
        else:
            token_var = ttk.Entry(token_frame, width=40)
            token_var.pack(side=tk.LEFT, padx=10)

        def next_step():
            if not "coordinator" in self.components:
                if not url_var.get():
                    messagebox.showerror("Required", "Coordinator URL required")
                    return
                self.coordinator_url = url_var.get()
                if not token_var.get():
                    messagebox.showerror("Required", "Auth token required")
                    return
                self.agent_token = token_var.get()

            self.agent_id = id_var.get()
            self.show_review()

        button_frame = ttk.Frame(frame)
        button_frame.pack(fill=tk.X, pady=20)

        ttk.Button(button_frame, text="Next >",
                  command=next_step).pack(side=tk.RIGHT)
        ttk.Button(button_frame, text="< Back",
                  command=self.show_component_selection).pack(side=tk.LEFT)

    def show_review(self):
        """Review and confirm installation"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Review Configuration",
                         font=("Arial", 16, "bold"))
        title.pack(pady=20)

        info = f"Components: {', '.join(self.components).title()}\n\n"

        if "coordinator" in self.components:
            info += f"Coordinator Port: {self.coordinator_port}\n"
            info += f"Admin Username: {self.admin_username}\n\n"

        if "agent" in self.components:
            info += f"Coordinator URL: {self.coordinator_url}\n"
            info += f"Agent ID: {self.agent_id}\n"

        info_label = ttk.Label(frame, text=info, justify=tk.LEFT,
                              font=("Arial", 10))
        info_label.pack(pady=20)

        confirm_label = ttk.Label(frame,
                                 text="Click 'Install' to proceed",
                                 font=("Arial", 10, "italic"))
        confirm_label.pack(pady=20)

        button_frame = ttk.Frame(frame)
        button_frame.pack(fill=tk.X, pady=20)

        ttk.Button(button_frame, text="Install",
                  command=self.install).pack(side=tk.RIGHT)
        ttk.Button(button_frame, text="< Back",
                  command=self.show_agent_config).pack(side=tk.LEFT)

    def install(self):
        """Perform installation"""
        self.clear_window()

        frame = ttk.Frame(self.root, padding="20")
        frame.pack(fill=tk.BOTH, expand=True)

        title = ttk.Label(frame, text="Installing...",
                         font=("Arial", 16, "bold"))
        title.pack(pady=20)

        progress = ttk.Progressbar(frame, length=400, mode="indeterminate")
        progress.pack(pady=20)
        progress.start()

        status = ttk.Label(frame, text="Preparing installation...",
                          font=("Arial", 10))
        status.pack(pady=10)

        def do_install():
            try:
                # Create install directory
                self.install_path.mkdir(parents=True, exist_ok=True)
                status.config(text="Creating directories...")
                self.root.update()

                # Create config directory
                self.config_path.mkdir(parents=True, exist_ok=True)
                status.config(text="Writing configuration...")
                self.root.update()

                # Write coordinator config if needed
                if "coordinator" in self.components:
                    self.write_coordinator_config()

                # Write agent config if needed
                if "agent" in self.components:
                    self.write_agent_config()

                status.config(text="Installing services...")
                self.root.update()

                # Install Windows services
                if "coordinator" in self.components:
                    self.install_service("coordinator")
                if "agent" in self.components:
                    self.install_service("agent")

                status.config(text="Opening dashboard...")
                self.root.update()

                # Open browser
                self.open_browser(f"http://localhost:{self.coordinator_port}")

                progress.stop()

                messagebox.showinfo("Success",
                    "ArcVault has been successfully installed!\n\n"
                    "Services are running and the dashboard is opening in your browser.")

                self.root.quit()

            except Exception as e:
                progress.stop()
                messagebox.showerror("Installation Error", str(e))
                self.show_review()

        # Run installation in background thread
        thread = threading.Thread(target=do_install)
        thread.daemon = True
        thread.start()

    def write_coordinator_config(self):
        """Write coordinator configuration"""
        config = {
            "port": self.coordinator_port,
            "admin": {
                "username": self.admin_username,
                "password": self.admin_password
            },
            "admin_token": self.admin_token
        }

        config_file = self.config_path / "coordinator-config.json"
        with open(config_file, "w") as f:
            json.dump(config, f, indent=2)

        # Restrict permissions
        os.chmod(config_file, 0o600)

    def write_agent_config(self):
        """Write agent configuration"""
        config = {
            "coordinator_url": self.coordinator_url,
            "agent_id": self.agent_id,
            "auth_token": self.agent_token
        }

        config_file = self.config_path / "agent-config.json"
        with open(config_file, "w") as f:
            json.dump(config, f, indent=2)

        # Restrict permissions
        os.chmod(config_file, 0o600)

    def install_service(self, service_type: str):
        """Install Windows service"""
        try:
            # Find the binary in PyInstaller's temp directory or current directory
            if getattr(sys, 'frozen', False):
                # Running as compiled .exe - binaries are bundled
                base_path = Path(sys._MEIPASS)
            else:
                # Running as script - binaries in dist/
                base_path = Path.cwd()

            if service_type == "coordinator":
                binary_name = "coordinator.exe"
                service_name = "ArcVaultCoordinator"
                display_name = "ArcVault Coordinator"
            elif service_type == "agent":
                binary_name = "agent.exe"
                service_name = "ArcVaultAgent"
                display_name = "ArcVault Agent"
            else:
                raise Exception(f"Unknown service type: {service_type}")

            # Find binary
            binary_src = base_path / binary_name
            if not binary_src.exists():
                raise Exception(f"Binary not found: {binary_src}")

            # Copy to install directory
            binary_dst = self.install_path / binary_name
            shutil.copy(binary_src, binary_dst)

            # Create Windows service using 'sc' command
            # sc create <name> binPath= <path> start= auto DisplayName= <display>
            result = subprocess.run(
                [
                    "sc", "create",
                    service_name,
                    f"binPath= {binary_dst}",
                    "start= auto",
                    f"DisplayName= {display_name}"
                ],
                capture_output=True,
                text=True,
                shell=True
            )

            if result.returncode != 0:
                # Check if service already exists
                if "already exists" in result.stderr:
                    pass  # Service already exists, that's OK
                else:
                    raise Exception(f"Failed to create service {service_name}: {result.stderr}")

            # Start the service
            start_result = subprocess.run(
                ["sc", "start", service_name],
                capture_output=True,
                text=True,
                shell=True
            )

            if start_result.returncode != 0:
                # Check if service is already running
                if "already running" not in start_result.stderr:
                    raise Exception(f"Failed to start service {service_name}: {start_result.stderr}")

        except Exception as e:
            raise Exception(f"Failed to install {service_type} service: {e}")

    def open_browser(self, url: str):
        """Open URL in default browser"""
        import webbrowser
        webbrowser.open(url)

    def generate_token(self, length: int) -> str:
        """Generate random token"""
        import secrets
        return secrets.token_hex(length // 2)

    def clear_window(self):
        """Clear all widgets from root window"""
        for widget in self.root.winfo_children():
            widget.destroy()


def main():
    """Entry point"""
    installer = ArcVaultInstaller()
    installer.run()


if __name__ == "__main__":
    main()
