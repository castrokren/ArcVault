#!/usr/bin/env python3
"""
Update Checker for ArcVault (Beta)
Handles checking for updates, downloading, and installing them
"""

import os
import sys
import requests
import subprocess
import tempfile
import logging
import hashlib
from typing import Optional, Dict, Any

# Public help URL for reporting issues
HELP_URL = "https://github.com/castrokren/arcvault-tool/issues/new/choose"


class UpdateChecker:
    def __init__(self):
        self.current_version = self.get_current_version()
        # GitHub repository URLs for ArcVault (Beta)
        self.update_url = (
            "https://api.github.com/repos/"
            "castrokren/arcvault-tool/releases/latest"
        )
        self.download_base_url = (
            "https://github.com/"
            "castrokren/arcvault-tool/releases/download"
        )
        # Stable manifest URL (optional, preferred when present)
        self.manifest_url_latest = (
            "https://github.com/"
            "castrokren/arcvault-tool/releases/latest/download/latest.json"
        )
        self.http_headers = {"User-Agent": "ArcVault-Updater"}
        self.logger = logging.getLogger(__name__)
        # Cached values from the last check to help downstream steps
        self._expected_sha256: Optional[str] = None
        self._silent_args: Optional[str] = None
        # Optional: expected Authenticode publisher subject (set via env)
        # Example value: CN=Your Company, O=...
        self._expected_publisher_subject: Optional[str] = os.environ.get(
            "ARCVault_EXPECTED_PUBLISHER"
        )

    def get_current_version(self):
        """Get current application version"""
        try:
            # 1) If running as frozen EXE, try reading version resource
            try:
                if getattr(sys, "frozen", False):
                    import win32api  # type: ignore

                    exe_path = sys.executable
                    info = win32api.GetFileVersionInfo(exe_path, "\\")
                    ms = info["FileVersionMS"]
                    ls = info["FileVersionLS"]
                    version_tuple = (
                        (ms >> 16) & 0xFFFF,
                        ms & 0xFFFF,
                        (ls >> 16) & 0xFFFF,
                        ls & 0xFFFF,
                    )
                    return ".".join(str(x) for x in version_tuple)
            except Exception:
                pass

            # 2) Parse VSVersionInfo-style version_info.txt (FileVersion)
            version_file = os.path.join(os.getcwd(), "version_info.txt")
            if os.path.exists(version_file):
                try:
                    with open(
                        version_file,
                        "r",
                        encoding="utf-8",
                        errors="ignore",
                    ) as f:
                        for line in f:
                            if "FileVersion" in line:
                                import re

                                match = re.search(
                                    r"'(?P<v>\d+\.\d+\.\d+\.\d+)'",
                                    line,
                                )
                                if match:
                                    return match.group("v")
                except Exception:
                    pass

            # 3) Legacy single-line format: version=...
            legacy_file = os.path.join(os.getcwd(), "version_info.txt")
            if os.path.exists(legacy_file):
                try:
                    with open(
                        legacy_file,
                        "r",
                        encoding="utf-8",
                        errors="ignore",
                    ) as f:
                        for line in f:
                            if line.startswith("version="):
                                return line.split("=")[1].strip()
                except Exception:
                    pass

            return "1.0.0.0"  # Default version
        except Exception as e:
            self.logger.error(f"Error reading version: {e}")
            return "1.0.0.0"

    def check_for_updates(self):
        """Check for available updates"""
        try:
            # Prefer manifest if available; fallback to GitHub Releases API
            manifest_info = self._fetch_manifest_latest()
            if manifest_info:
                latest_version = manifest_info.get("version", "").strip()
                asset = self._select_preferred_asset(
                    manifest_info.get("assets") or []
                )
                download_url = asset.get("url") if asset else None
                asset_name = asset.get("name") if asset else None
                self._expected_sha256 = asset.get("sha256") if asset else None
                self._silent_args = asset.get("silent_args") if asset else None
                self.logger.debug(
                    "Manifest parsed: version=%s asset=%s url=%s",
                    latest_version,
                    asset_name,
                    download_url,
                )

                if (
                    latest_version
                    and self.compare_versions(
                        latest_version, self.current_version
                    )
                    > 0
                ):
                    return {
                        "available": True,
                        "version": latest_version,
                        "download_url": download_url,
                        "asset_name": asset_name,
                        "release_notes": "",
                        "release_page": manifest_info.get(
                            "notes_url"
                        ),
                        "error": (
                            None if download_url else "No asset in manifest"
                        ),
                        "sha256": self._expected_sha256,
                        "help_url": HELP_URL,
                    }

            # Check GitHub releases for updates (fallback)
            response = requests.get(
                self.update_url,
                timeout=10,
                headers=self.http_headers,
            )
            if response.status_code != 200:
                self.logger.error(
                    "GitHub API error: status=%s url=%s",
                    response.status_code,
                    self.update_url,
                )
            if response.status_code == 200:
                release_data = response.json()
                latest_version = release_data.get("tag_name", "").lstrip("v")

                # Try to determine the correct asset to download
                assets = release_data.get("assets", []) or []
                download_url = None
                asset_name = None

                # Prefer MSI assets
                for a in assets:
                    name = (a.get("name") or "").lower()
                    if name.endswith(".msi"):
                        download_url = a.get("browser_download_url")
                        asset_name = a.get("name")
                        break

                # Fallback to EXE assets
                if not download_url:
                    for a in assets:
                        name = (a.get("name") or "").lower()
                        if name.endswith(".exe"):
                            download_url = a.get("browser_download_url")
                            asset_name = a.get("name")
                            break

                if (
                    self.compare_versions(latest_version, self.current_version)
                    > 0
                ):
                    self.logger.debug(
                        "Release parsed: version=%s asset=%s url=%s",
                        latest_version,
                        asset_name,
                        download_url,
                    )
                    # Reset checksum/silent args in fallback mode
                    self._expected_sha256 = None
                    self._silent_args = None
                    return {
                        "available": True,
                        "version": latest_version,
                        "download_url": download_url,
                        "asset_name": asset_name,
                        "release_notes": release_data.get("body", ""),
                        "release_page": release_data.get("html_url"),
                        "error": (
                            None
                            if download_url
                            else "No suitable asset found in release"
                        ),
                        "sha256": None,
                        "help_url": HELP_URL,
                    }

            return {
                "available": False,
                "version": self.current_version,
                "error": None,
                "help_url": HELP_URL,
            }

        except Exception as e:
            self.logger.error(f"Error checking for updates: {e}")
            return {
                "available": False,
                "version": self.current_version,
                "error": str(e),
                "help_url": HELP_URL,
            }

    def compare_versions(self, version1, version2):
        """Compare two version strings"""
        try:
            v1_parts = [int(x) for x in version1.split(".")]
            v2_parts = [int(x) for x in version2.split(".")]

            for i in range(max(len(v1_parts), len(v2_parts))):
                v1_part = v1_parts[i] if i < len(v1_parts) else 0
                v2_part = v2_parts[i] if i < len(v2_parts) else 0

                if v1_part > v2_part:
                    return 1
                elif v1_part < v2_part:
                    return -1

            return 0
        except Exception as e:
            self.logger.error(f"Error comparing versions: {e}")
            return 0

    def download_update(self, download_url):
        """Download the update file"""
        try:
            is_exe = (download_url or "").lower().endswith(".exe")
            suffix = ".exe" if is_exe else ".msi"
            self.logger.info("Downloading update: %s", download_url)
            response = requests.get(
                download_url,
                stream=True,
                timeout=30,
            )
            if response.status_code != 200:
                self.logger.error(
                    "Download failed: status=%s url=%s",
                    response.status_code,
                    download_url,
                )
                response.raise_for_status()

            # Create temporary file
            temp_file = tempfile.NamedTemporaryFile(
                delete=False,
                suffix=suffix,
            )
            self.logger.debug("Writing to temp file: %s", temp_file.name)

            # Download the file
            hasher = hashlib.sha256()
            for chunk in response.iter_content(chunk_size=8192):
                if not chunk:
                    continue
                temp_file.write(chunk)
                hasher.update(chunk)

            temp_file.close()
            # Verify checksum if provided
            if self._expected_sha256:
                actual = hasher.hexdigest().lower()
                expected = self._expected_sha256.lower().strip()
                if actual != expected:
                    try:
                        os.unlink(temp_file.name)
                    except Exception:
                        pass
                    self.logger.error(
                        "SHA-256 mismatch: expected %s, got %s",
                        expected,
                        actual,
                    )
                    return None
                else:
                    self.logger.debug("SHA-256 verified: %s", actual)

            # Verify Authenticode signature if configured and on Windows
            if os.name == "nt" and self._expected_publisher_subject:
                if not self._verify_signature(
                    temp_file.name, self._expected_publisher_subject
                ):
                    try:
                        os.unlink(temp_file.name)
                    except Exception:
                        pass
                    self.logger.error(
                        "Signature verification failed for %s", temp_file.name
                    )
                    return None
                else:
                    self.logger.debug(
                        "Signature verified for %s", temp_file.name
                    )

            return temp_file.name

        except Exception as e:
            self.logger.error(
                "Error downloading update from %s: %s", download_url, e
            )
            return None

    def install_update(self, update_file):
        """Install the downloaded update"""
        try:
            # For MSI files, use msiexec
            if update_file.endswith(".msi"):
                args = ["msiexec", "/i", update_file]
                if self._silent_args:
                    # Allow manifest-provided silent args
                    args.extend(self._silent_args.split())
                else:
                    args.extend(["/quiet", "/norestart"])
                self.logger.info("Running MSI installer: %s", " ".join(args))
                result = subprocess.run(
                    args,
                    capture_output=True,
                    text=True,
                )

                if result.returncode == 0:
                    return True
                else:
                    self.logger.error(
                        "MSI install failed: code=%s stderr=%s",
                        result.returncode,
                        (result.stderr or "").strip(),
                    )
                    return False

            # For EXE installers, attempt silent install
            # If it fails, try normal run
            if update_file.endswith(".exe"):
                # Try manifest-provided silent args first if present
                if self._silent_args:
                    try:
                        self.logger.info(
                            "Running EXE installer with args: %s",
                            self._silent_args,
                        )
                        result = subprocess.run(
                            [update_file] + self._silent_args.split(),
                            capture_output=True,
                            text=True,
                        )
                        if result.returncode == 0:
                            return True
                    except Exception:
                        pass
                # Try common silent flags first
                for args in (
                    [update_file, "/S"],
                    [update_file, "/silent"],
                    [update_file, "/quiet"],
                    [update_file],
                ):
                    try:
                        self.logger.debug("Trying EXE args: %s", args)
                        result = subprocess.run(
                            args,
                            capture_output=True,
                            text=True,
                        )
                        # Consider returncode 0 as success
                        if result.returncode == 0:
                            return True
                    except Exception:
                        continue
                self.logger.error(
                    "EXE install failed or no supported silent flags"
                )
                return False

            return False

        except Exception as e:
            self.logger.error(f"Error installing update: {e}")
            return False
        finally:
            # Clean up temporary file
            try:
                os.unlink(update_file)
            except Exception:
                pass

    def perform_notepad_style_update(self, update_info):
        """Perform Notepad++-style update (download and prepare for restart)"""
        try:
            if not update_info.get("available"):
                return False

            # Download the update
            update_file = self.download_update(update_info["download_url"])
            if not update_file:
                return False

            # Create restart script
            restart_script = self.create_restart_script(update_file)
            if not restart_script:
                return False

            return True

        except Exception as e:
            self.logger.error(f"Error performing update: {e}")
            return False

    def create_restart_script(self, update_file):
        """Create a script to restart the application after update"""
        try:
            script_content = f"""@echo off
echo Waiting for application to close...
timeout /t 2 /nobreak >nul

echo Installing update...
msiexec /i "{update_file}" /quiet /norestart

echo Starting application...
start "" "{sys.executable}" "{sys.argv[0]}"

del "%~f0"
"""

            script_file = tempfile.NamedTemporaryFile(
                delete=False, suffix=".bat", mode="w"
            )
            script_file.write(script_content)
            script_file.close()

            # Execute the restart script
            subprocess.Popen([script_file.name], shell=True)

            return script_file.name

        except Exception as e:
            self.logger.error(f"Error creating restart script: {e}")
            return None

    # Internal helpers
    def _fetch_manifest_latest(self) -> Optional[Dict[str, Any]]:
        try:
            response = requests.get(
                self.manifest_url_latest, timeout=10, headers=self.http_headers
            )
            if response.status_code == 200:
                return response.json()
            else:
                self.logger.debug(
                    "Manifest fetch returned %s for %s",
                    response.status_code,
                    self.manifest_url_latest,
                )
            return None
        except Exception as e:
            self.logger.debug("Manifest fetch failed: %s", e)
            return None

    def _select_preferred_asset(self, assets: Any) -> Optional[Dict[str, Any]]:
        try:
            # Prefer MSI, fallback to EXE
            for a in assets:
                name = (a.get("name") or "").lower()
                if name.endswith(".msi"):
                    return a
            for a in assets:
                name = (a.get("name") or "").lower()
                if name.endswith(".exe"):
                    return a
            return None
        except Exception:
            return None

    def _verify_signature(
        self,
        file_path: str,
        expected_subject_substr: str,
    ) -> bool:
        try:
            if os.name != "nt":
                return True
            # Query Authenticode signature via PowerShell
            ps_cmd = "".join(
                [
                    "$s=Get-AuthenticodeSignature -FilePath '",
                    file_path,
                    "';",
                    "$s.Status.ToString();",
                    "$s.SignerCertificate.Subject",
                ]
            )
            result = subprocess.run(
                [
                    "powershell",
                    "-NoProfile",
                    "-ExecutionPolicy",
                    "Bypass",
                    "-Command",
                    ps_cmd,
                ],
                capture_output=True,
                text=True,
            )
            if result.returncode != 0:
                return False
            lines = [
                line.strip()
                for line in (result.stdout or "").splitlines()
                if line.strip()
            ]
            if not lines:
                return False
            status = lines[0]
            subject = lines[1] if len(lines) > 1 else ""
            self.logger.debug("Authenticode status=%s subject=%s", status, subject)
            if status.lower() != "valid":
                return False
            if (
                expected_subject_substr
                and expected_subject_substr not in subject
            ):
                return False
            return True
        except Exception as e:
            self.logger.error("Signature verification error: %s", e)
            return False
