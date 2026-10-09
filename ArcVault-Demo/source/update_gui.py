#!/usr/bin/env python3
"""
GUI Update Dialog for ArcVault (Beta)
Provides a user-friendly interface for update notifications and installation
"""

import tkinter as tk
from tkinter import ttk, messagebox
import threading
import sys
import os
import webbrowser
import ctypes

# Add current directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from update_checker import UpdateChecker, HELP_URL


class UpdateDialog:
    def __init__(self, parent=None):
        self.parent = parent
        self.checker = UpdateChecker()
        self.update_info = None
        self.result = None

    def check_for_updates(self):
        """Check for updates and show dialog if available"""
        try:
            self.update_info = self.checker.check_for_updates()
            if self.update_info.get("available"):
                self.show_update_dialog()
                return self.result
            return None
        except Exception as e:
            print(f"Error checking for updates: {e}")
            return None

    def show_update_dialog(self):
        """Show the update dialog"""
        # Create the dialog window
        dialog = tk.Toplevel(self.parent) if self.parent else tk.Tk()
        dialog.title("ArcVault (Beta) - Update Available")
        dialog.geometry("500x400")
        dialog.resizable(False, False)

        # Center the dialog
        dialog.transient(self.parent) if self.parent else None
        dialog.grab_set()

        # Make dialog modal
        dialog.focus_set()
        dialog.wait_window()

        # Configure style
        style = ttk.Style()
        style.theme_use("clam")

        # Main frame
        main_frame = ttk.Frame(dialog, padding="20")
        main_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))

        # Title
        title_label = ttk.Label(
            main_frame, text="🔄 Update Available!", font=("Arial", 16, "bold")
        )
        title_label.grid(row=0, column=0, columnspan=2, pady=(0, 10))

        # Version info
        version_frame = ttk.LabelFrame(
            main_frame, text="Version Information", padding="10"
        )
        version_frame.grid(
            row=1, column=0, columnspan=2, sticky=(tk.W, tk.E), pady=(0, 10)
        )

        ttk.Label(
            version_frame, text=f"Current Version: {self.checker.current_version}"
        ).grid(row=0, column=0, sticky=tk.W)
        ttk.Label(
            version_frame, text=f"New Version: {self.update_info['version']}"
        ).grid(row=1, column=0, sticky=tk.W)

        # Release notes
        if self.update_info.get("release_notes"):
            notes_frame = ttk.LabelFrame(main_frame, text="Release Notes", padding="10")
            notes_frame.grid(
                row=2,
                column=0,
                columnspan=2,
                sticky=(tk.W, tk.E, tk.N, tk.S),
                pady=(0, 10),
            )

            # Create text widget with scrollbar
            text_frame = ttk.Frame(notes_frame)
            text_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))

            text_widget = tk.Text(text_frame, height=8, width=60, wrap=tk.WORD)
            scrollbar = ttk.Scrollbar(
                text_frame, orient=tk.VERTICAL, command=text_widget.yview
            )
            text_widget.configure(yscrollcommand=scrollbar.set)

            text_widget.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
            scrollbar.grid(row=0, column=1, sticky=(tk.N, tk.S))

            text_widget.insert(tk.END, self.update_info["release_notes"])
            text_widget.config(state=tk.DISABLED)

            # Configure grid weights
            notes_frame.columnconfigure(0, weight=1)
            notes_frame.rowconfigure(0, weight=1)
            text_frame.columnconfigure(0, weight=1)
            text_frame.rowconfigure(0, weight=1)

        # Buttons frame
        button_frame = ttk.Frame(main_frame)
        button_frame.grid(row=3, column=0, columnspan=2, pady=(10, 0))

        # Update button
        update_btn = ttk.Button(
            button_frame,
            text="📥 Install Update",
            command=lambda: self.install_update(dialog),
        )
        update_btn.grid(row=0, column=0, padx=(0, 10))

        # Release page button (if available)
        release_url = self.update_info.get("release_page")
        if release_url:
            view_btn = ttk.Button(
                button_frame,
                text="📝 View Release Page",
                command=lambda u=release_url: webbrowser.open(u),
            )
            view_btn.grid(row=0, column=1, padx=(10, 0))

        # Report Issue button
        report_btn = ttk.Button(
            button_frame,
            text="🐞 Report Issue",
            command=lambda: webbrowser.open(self.update_info.get("help_url") or HELP_URL),
        )
        report_btn.grid(row=0, column=2, padx=(10, 0))

        # Skip button
        skip_btn = ttk.Button(
            button_frame,
            text="⏭️ Skip for Now",
            command=lambda: self.skip_update(dialog),
        )
        skip_btn.grid(row=0, column=3, padx=(10, 0))

        # Later button
        later_btn = ttk.Button(
            button_frame,
            text="⏰ Remind Me Later",
            command=lambda: self.remind_later(dialog),
        )
        later_btn.grid(row=0, column=4, padx=(10, 0))

        # Configure grid weights
        main_frame.columnconfigure(0, weight=1)
        main_frame.rowconfigure(2, weight=1)

        # Center the dialog on screen
        dialog.update_idletasks()
        x = (dialog.winfo_screenwidth() // 2) - (dialog.winfo_width() // 2)
        y = (dialog.winfo_screenheight() // 2) - (dialog.winfo_height() // 2)
        dialog.geometry(f"+{x}+{y}")

        # Wait for user response
        dialog.wait_window()

    def install_update(self, dialog):
        """Install the update"""
        try:
            # Show progress dialog
            progress_dialog = tk.Toplevel(dialog)
            progress_dialog.title("Installing Update")
            progress_dialog.geometry("400x150")
            progress_dialog.transient(dialog)
            progress_dialog.grab_set()

            # Center progress dialog
            progress_dialog.update_idletasks()
            x = (progress_dialog.winfo_screenwidth() // 2) - (
                progress_dialog.winfo_width() // 2
            )
            y = (progress_dialog.winfo_screenheight() // 2) - (
                progress_dialog.winfo_height() // 2
            )
            progress_dialog.geometry(f"+{x}+{y}")

            # Progress frame
            progress_frame = ttk.Frame(progress_dialog, padding="20")
            progress_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))

            # Status label
            status_label = ttk.Label(progress_frame, text="Downloading update...")
            status_label.grid(row=0, column=0, pady=(0, 10))

            # Progress bar
            progress_bar = ttk.Progressbar(progress_frame, mode="indeterminate")
            progress_bar.grid(row=1, column=0, sticky=(tk.W, tk.E), pady=(0, 10))
            progress_bar.start()

            # Configure grid weights
            progress_frame.columnconfigure(0, weight=1)
            progress_dialog.columnconfigure(0, weight=1)
            progress_dialog.rowconfigure(0, weight=1)

            def update_process():
                """Process the update in a separate thread"""
                try:
                    # Download update
                    status_label.config(text="Downloading update...")
                    progress_dialog.update()

                    update_file = self.checker.download_update(
                        self.update_info["download_url"]
                    )

                    if update_file:
                        status_label.config(text="Installing update...")
                        progress_dialog.update()

                        # Admin hint on Windows
                        try:
                            if os.name == "nt":
                                if not ctypes.windll.shell32.IsUserAnAdmin():
                                    status_label.config(
                                        text=(
                                            "Installing update (note: admin "
                                            "rights may be required)..."
                                        )
                                    )
                                    progress_dialog.update()
                        except Exception:
                            pass

                        if self.checker.install_update(update_file):
                            status_label.config(text="Update installed successfully!")
                            progress_bar.stop()
                            progress_dialog.after(
                                2000,
                                lambda: self.complete_update(progress_dialog, True),
                            )
                        else:
                            status_label.config(
                                text=(
                                    "Error installing update. If this "
                                    "persists, try running as Administrator."
                                )
                            )
                            progress_bar.stop()
                            progress_dialog.after(
                                2000,
                                lambda: self.complete_update(progress_dialog, False),
                            )
                    else:
                        status_label.config(
                            text=(
                                "Error downloading or verifying update. "
                                "Please check your network and try again."
                            )
                        )
                        progress_bar.stop()
                        progress_dialog.after(
                            2000, lambda: self.complete_update(progress_dialog, False)
                        )

                except Exception as e:
                    status_label.config(text=f"Error: {str(e)}")
                    progress_bar.stop()
                    progress_dialog.after(
                        2000, lambda: self.complete_update(progress_dialog, False)
                    )

            # Start update process in separate thread
            update_thread = threading.Thread(target=update_process)
            update_thread.daemon = True
            update_thread.start()

        except Exception as e:
            messagebox.showerror("Update Error", f"Error during update: {str(e)}")
            dialog.destroy()

    def complete_update(self, progress_dialog, success):
        """Complete the update process"""
        progress_dialog.destroy()

        if success:
            messagebox.showinfo(
                "Update Complete",
                "Update installed successfully!\n\nPlease restart the application.",
            )
            self.result = "installed"
        else:
            # Build helpful failure message with link
            help_url = (self.update_info or {}).get("help_url") or HELP_URL
            message = (
                "Failed to install the update.\n\n"
                "Try running as Administrator or check your network.\n\n"
                "You can report this issue here:\n"
                f"{help_url}"
            )
            messagebox.showerror("Update Failed", message)
            self.result = "failed"

    def skip_update(self, dialog):
        """Skip the update"""
        self.result = "skipped"
        dialog.destroy()

    def remind_later(self, dialog):
        """Remind later"""
        self.result = "remind_later"
        dialog.destroy()


def check_for_updates_gui(parent=None):
    """Check for updates and show GUI dialog if available"""
    update_dialog = UpdateDialog(parent)
    return update_dialog.check_for_updates()


if __name__ == "__main__":
    # Test the update dialog
    result = check_for_updates_gui()
    print(f"Update result: {result}")
