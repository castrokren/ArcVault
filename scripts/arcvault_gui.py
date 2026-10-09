#!/usr/bin/env python3
"""
ArcVault Unified GUI Application

A comprehensive GUI tool that combines:
- SSL Certificate Generation
- JWT Key Generation  
- Client Installation Management

This replaces the individual command-line executables with a user-friendly interface.
"""

import tkinter as tk
from tkinter import ttk, filedialog, messagebox, scrolledtext
import os
import sys
import subprocess
import threading
from pathlib import Path
from datetime import datetime, timedelta
import tempfile
import shutil

# Import the certificate and JWT generation modules
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))

try:
    from cryptography import x509
    from cryptography.x509.oid import NameOID
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
    import ipaddress
except ImportError:
    messagebox.showerror("Missing Dependencies", 
                        "Required cryptography library not found.\n"
                        "Please install it with: pip install cryptography")
    sys.exit(1)


class ArcVaultGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("ArcVault Management Suite")
        self.root.geometry("800x600")
        self.root.minsize(700, 500)
        
        # Configure style
        style = ttk.Style()
        style.theme_use('clam')
        
        # Create main notebook for tabs
        self.notebook = ttk.Notebook(root)
        self.notebook.pack(fill=tk.BOTH, expand=True, padx=10, pady=10)
        
        # Create tabs
        self.create_cert_tab()
        self.create_jwt_tab()
        self.create_installer_tab()
        self.create_about_tab()
        
        # Status bar
        self.status_var = tk.StringVar()
        self.status_var.set("Ready")
        status_bar = ttk.Label(root, textvariable=self.status_var, relief=tk.SUNKEN)
        status_bar.pack(side=tk.BOTTOM, fill=tk.X)

    def create_cert_tab(self):
        """Create SSL Certificate Generation tab"""
        cert_frame = ttk.Frame(self.notebook)
        self.notebook.add(cert_frame, text="SSL Certificates")
        
        # Main container with scrollbar
        canvas = tk.Canvas(cert_frame)
        scrollbar = ttk.Scrollbar(cert_frame, orient="vertical", command=canvas.yview)
        scrollable_frame = ttk.Frame(canvas)
        
        scrollable_frame.bind(
            "<Configure>",
            lambda e: canvas.configure(scrollregion=canvas.bbox("all"))
        )
        
        canvas.create_window((0, 0), window=scrollable_frame, anchor="nw")
        canvas.configure(yscrollcommand=scrollbar.set)
        
        # Title
        title = ttk.Label(scrollable_frame, text="SSL Certificate Generator", 
                         font=('Arial', 14, 'bold'))
        title.pack(pady=(0, 20))
        
        # Output Directory
        dir_frame = ttk.LabelFrame(scrollable_frame, text="Output Directory", padding=10)
        dir_frame.pack(fill=tk.X, padx=20, pady=5)
        
        self.cert_output_var = tk.StringVar(value=str(Path.cwd() / "keys"))
        ttk.Entry(dir_frame, textvariable=self.cert_output_var, width=50).pack(side=tk.LEFT, fill=tk.X, expand=True)
        ttk.Button(dir_frame, text="Browse", command=self.browse_cert_output).pack(side=tk.RIGHT, padx=(5, 0))
        
        # Certificate Settings
        settings_frame = ttk.LabelFrame(scrollable_frame, text="Certificate Settings", padding=10)
        settings_frame.pack(fill=tk.X, padx=20, pady=5)
        
        # Hostname
        ttk.Label(settings_frame, text="Hostname:").grid(row=0, column=0, sticky=tk.W, pady=2)
        self.cert_hostname_var = tk.StringVar(value="localhost")
        ttk.Entry(settings_frame, textvariable=self.cert_hostname_var, width=30).grid(row=0, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        # Key Size
        ttk.Label(settings_frame, text="Key Size:").grid(row=1, column=0, sticky=tk.W, pady=2)
        self.cert_keysize_var = tk.StringVar(value="2048")
        keysize_combo = ttk.Combobox(settings_frame, textvariable=self.cert_keysize_var, 
                                    values=["2048", "3072", "4096"], state="readonly", width=10)
        keysize_combo.grid(row=1, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        # Validity Days
        ttk.Label(settings_frame, text="Valid for (days):").grid(row=2, column=0, sticky=tk.W, pady=2)
        self.cert_days_var = tk.StringVar(value="365")
        ttk.Entry(settings_frame, textvariable=self.cert_days_var, width=10).grid(row=2, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        # Advanced Options
        advanced_frame = ttk.LabelFrame(scrollable_frame, text="Advanced Options", padding=10)
        advanced_frame.pack(fill=tk.X, padx=20, pady=5)
        
        self.cert_force_var = tk.BooleanVar()
        ttk.Checkbutton(advanced_frame, text="Overwrite existing certificates", 
                       variable=self.cert_force_var).pack(anchor=tk.W)
        
        # Subject Information
        subject_frame = ttk.LabelFrame(scrollable_frame, text="Certificate Subject", padding=10)
        subject_frame.pack(fill=tk.X, padx=20, pady=5)
        
        ttk.Label(subject_frame, text="Country (2 letters):").grid(row=0, column=0, sticky=tk.W, pady=2)
        self.cert_country_var = tk.StringVar(value="US")
        ttk.Entry(subject_frame, textvariable=self.cert_country_var, width=5).grid(row=0, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        ttk.Label(subject_frame, text="Organization:").grid(row=1, column=0, sticky=tk.W, pady=2)
        self.cert_org_var = tk.StringVar(value="ArcVault")
        ttk.Entry(subject_frame, textvariable=self.cert_org_var, width=30).grid(row=1, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        # Generate Button
        button_frame = ttk.Frame(scrollable_frame)
        button_frame.pack(pady=20)
        
        ttk.Button(button_frame, text="Generate SSL Certificate", 
                  command=self.generate_certificate, style='Accent.TButton').pack()
        
        # Output Log
        log_frame = ttk.LabelFrame(scrollable_frame, text="Output", padding=10)
        log_frame.pack(fill=tk.BOTH, expand=True, padx=20, pady=5)
        
        self.cert_log = scrolledtext.ScrolledText(log_frame, height=8, width=70)
        self.cert_log.pack(fill=tk.BOTH, expand=True)
        
        # Pack canvas and scrollbar
        canvas.pack(side="left", fill="both", expand=True)
        scrollbar.pack(side="right", fill="y")

    def create_jwt_tab(self):
        """Create JWT Key Generation tab"""
        jwt_frame = ttk.Frame(self.notebook)
        self.notebook.add(jwt_frame, text="JWT Keys")
        
        # Title
        title = ttk.Label(jwt_frame, text="JWT Key Pair Generator", 
                         font=('Arial', 14, 'bold'))
        title.pack(pady=(20, 20))
        
        # Output Directory
        dir_frame = ttk.LabelFrame(jwt_frame, text="Output Directory", padding=10)
        dir_frame.pack(fill=tk.X, padx=20, pady=5)
        
        self.jwt_output_var = tk.StringVar(value=str(Path.cwd() / "keys"))
        ttk.Entry(dir_frame, textvariable=self.jwt_output_var, width=50).pack(side=tk.LEFT, fill=tk.X, expand=True)
        ttk.Button(dir_frame, text="Browse", command=self.browse_jwt_output).pack(side=tk.RIGHT, padx=(5, 0))
        
        # JWT Settings
        settings_frame = ttk.LabelFrame(jwt_frame, text="Key Settings", padding=10)
        settings_frame.pack(fill=tk.X, padx=20, pady=5)
        
        # Key Size
        ttk.Label(settings_frame, text="Key Size:").grid(row=0, column=0, sticky=tk.W, pady=5)
        self.jwt_keysize_var = tk.StringVar(value="2048")
        keysize_combo = ttk.Combobox(settings_frame, textvariable=self.jwt_keysize_var, 
                                    values=["2048", "3072", "4096"], state="readonly", width=10)
        keysize_combo.grid(row=0, column=1, sticky=tk.W, padx=(5, 0), pady=5)
        
        # Advanced Options
        advanced_frame = ttk.LabelFrame(jwt_frame, text="Options", padding=10)
        advanced_frame.pack(fill=tk.X, padx=20, pady=5)
        
        self.jwt_force_var = tk.BooleanVar()
        ttk.Checkbutton(advanced_frame, text="Overwrite existing keys", 
                       variable=self.jwt_force_var).pack(anchor=tk.W)
        
        # Generate Button
        button_frame = ttk.Frame(jwt_frame)
        button_frame.pack(pady=20)
        
        ttk.Button(button_frame, text="Generate JWT Key Pair", 
                  command=self.generate_jwt_keys, style='Accent.TButton').pack()
        
        # Output Log
        log_frame = ttk.LabelFrame(jwt_frame, text="Output", padding=10)
        log_frame.pack(fill=tk.BOTH, expand=True, padx=20, pady=5)
        
        self.jwt_log = scrolledtext.ScrolledText(log_frame, height=12, width=70)
        self.jwt_log.pack(fill=tk.BOTH, expand=True)

    def create_installer_tab(self):
        """Create Client Installer Management tab"""
        installer_frame = ttk.Frame(self.notebook)
        self.notebook.add(installer_frame, text="Client Installer")
        
        # Title
        title = ttk.Label(installer_frame, text="ArcVault Client Management", 
                         font=('Arial', 14, 'bold'))
        title.pack(pady=(20, 20))
        
        # Installation Actions
        actions_frame = ttk.LabelFrame(installer_frame, text="Installation Actions", padding=15)
        actions_frame.pack(fill=tk.X, padx=20, pady=5)
        
        # Run Client Installer
        installer_btn_frame = ttk.Frame(actions_frame)
        installer_btn_frame.pack(fill=tk.X, pady=5)
        
        ttk.Button(installer_btn_frame, text="Run Client Installer", 
                  command=self.run_client_installer).pack(side=tk.LEFT)
        ttk.Label(installer_btn_frame, text="Install ArcVault client on this machine").pack(side=tk.LEFT, padx=(10, 0))
        
        # Service Management
        service_frame = ttk.LabelFrame(installer_frame, text="Service Management", padding=15)
        service_frame.pack(fill=tk.X, padx=20, pady=5)
        
        # Service control buttons
        service_btn_frame = ttk.Frame(service_frame)
        service_btn_frame.pack(fill=tk.X, pady=5)
        
        ttk.Button(service_btn_frame, text="Install Service", 
                  command=lambda: self.service_action("install")).pack(side=tk.LEFT, padx=(0, 5))
        ttk.Button(service_btn_frame, text="Start Service", 
                  command=lambda: self.service_action("start")).pack(side=tk.LEFT, padx=5)
        ttk.Button(service_btn_frame, text="Stop Service", 
                  command=lambda: self.service_action("stop")).pack(side=tk.LEFT, padx=5)
        ttk.Button(service_btn_frame, text="Uninstall Service", 
                  command=lambda: self.service_action("uninstall")).pack(side=tk.LEFT, padx=5)
        
        # Service status
        status_btn_frame = ttk.Frame(service_frame)
        status_btn_frame.pack(fill=tk.X, pady=5)
        
        ttk.Button(status_btn_frame, text="Check Service Status", 
                  command=self.check_service_status).pack(side=tk.LEFT)
        ttk.Button(status_btn_frame, text="Test Agent Connection", 
                  command=self.test_agent_connection).pack(side=tk.LEFT, padx=(10, 0))
        
        # Quick Setup
        setup_frame = ttk.LabelFrame(installer_frame, text="Quick Setup", padding=15)
        setup_frame.pack(fill=tk.X, padx=20, pady=5)
        
        ttk.Label(setup_frame, text="Server URL:").grid(row=0, column=0, sticky=tk.W, pady=2)
        self.server_url_var = tk.StringVar(value="https://backup-server.company.com:8443")
        ttk.Entry(setup_frame, textvariable=self.server_url_var, width=40).grid(row=0, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        ttk.Label(setup_frame, text="Auth Token:").grid(row=1, column=0, sticky=tk.W, pady=2)
        self.auth_token_var = tk.StringVar()
        ttk.Entry(setup_frame, textvariable=self.auth_token_var, width=40, show="*").grid(row=1, column=1, sticky=tk.W, padx=(5, 0), pady=2)
        
        ttk.Button(setup_frame, text="Configure Agent", 
                  command=self.configure_agent).grid(row=2, column=0, columnspan=2, pady=10)
        
        # Output Log
        log_frame = ttk.LabelFrame(installer_frame, text="Output", padding=10)
        log_frame.pack(fill=tk.BOTH, expand=True, padx=20, pady=5)
        
        self.installer_log = scrolledtext.ScrolledText(log_frame, height=10, width=70)
        self.installer_log.pack(fill=tk.BOTH, expand=True)

    def create_about_tab(self):
        """Create About tab"""
        about_frame = ttk.Frame(self.notebook)
        self.notebook.add(about_frame, text="About")
        
        # Center content
        content_frame = ttk.Frame(about_frame)
        content_frame.pack(expand=True, fill=tk.BOTH)
        
        # Title
        title = ttk.Label(content_frame, text="ArcVault Management Suite", 
                         font=('Arial', 16, 'bold'))
        title.pack(pady=(50, 10))
        
        # Version
        version = ttk.Label(content_frame, text="Version 1.0.0", 
                           font=('Arial', 12))
        version.pack(pady=5)
        
        # Description
        desc = ttk.Label(content_frame, text="Unified management tool for ArcVault backup system", 
                        font=('Arial', 10))
        desc.pack(pady=5)
        
        # Features
        features_frame = ttk.LabelFrame(content_frame, text="Features", padding=20)
        features_frame.pack(pady=20, padx=50, fill=tk.X)
        
        features = [
            "• SSL Certificate Generation",
            "• JWT Key Pair Generation", 
            "• Client Installation Management",
            "• Windows Service Control",
            "• Agent Configuration",
            "• Connection Testing"
        ]
        
        for feature in features:
            ttk.Label(features_frame, text=feature).pack(anchor=tk.W, pady=2)
        
        # Support
        support_frame = ttk.Frame(content_frame)
        support_frame.pack(pady=20)
        
        ttk.Label(support_frame, text="Support: https://github.com/castrokren/arcvault-tool", 
                 font=('Arial', 10)).pack()

    # Event handlers
    def browse_cert_output(self):
        directory = filedialog.askdirectory(initialdir=self.cert_output_var.get())
        if directory:
            self.cert_output_var.set(directory)

    def browse_jwt_output(self):
        directory = filedialog.askdirectory(initialdir=self.jwt_output_var.get())
        if directory:
            self.jwt_output_var.set(directory)

    def log_to_widget(self, widget, message):
        """Thread-safe logging to text widget"""
        def append():
            widget.insert(tk.END, f"[{datetime.now().strftime('%H:%M:%S')}] {message}\n")
            widget.see(tk.END)
            widget.update()
        
        if threading.current_thread() == threading.main_thread():
            append()
        else:
            widget.after(0, append)

    def generate_certificate(self):
        """Generate SSL certificate in background thread"""
        def worker():
            try:
                self.status_var.set("Generating SSL certificate...")
                self.log_to_widget(self.cert_log, "Starting SSL certificate generation...")
                
                output_dir = Path(self.cert_output_var.get())
                hostname = self.cert_hostname_var.get().strip()
                key_size = int(self.cert_keysize_var.get())
                days = int(self.cert_days_var.get())
                force = self.cert_force_var.get()
                country = self.cert_country_var.get().strip()
                org = self.cert_org_var.get().strip()
                
                # Validate inputs
                if not hostname:
                    raise ValueError("Hostname is required")
                if not country or len(country) != 2:
                    raise ValueError("Country must be exactly 2 letters")
                if not org:
                    raise ValueError("Organization is required")
                
                # Create output directory
                output_dir.mkdir(parents=True, exist_ok=True)
                
                # Check for existing files
                cert_file = output_dir / "server.crt"
                key_file = output_dir / "server.key"
                ca_file = output_dir / "ca-bundle.crt"
                
                if not force and any(f.exists() for f in [cert_file, key_file, ca_file]):
                    if not messagebox.askyesno("Files Exist", 
                                             "Certificate files already exist. Overwrite?"):
                        self.status_var.set("Certificate generation cancelled")
                        return
                
                self.log_to_widget(self.cert_log, f"Generating {key_size}-bit RSA key pair...")
                
                # Generate private key
                private_key = rsa.generate_private_key(
                    public_exponent=65537,
                    key_size=key_size,
                )
                
                # Create certificate
                subject = issuer = x509.Name([
                    x509.NameAttribute(NameOID.COUNTRY_NAME, country),
                    x509.NameAttribute(NameOID.ORGANIZATION_NAME, org),
                    x509.NameAttribute(NameOID.COMMON_NAME, hostname),
                ])
                
                cert = x509.CertificateBuilder().subject_name(
                    subject
                ).issuer_name(
                    issuer
                ).public_key(
                    private_key.public_key()
                ).serial_number(
                    x509.random_serial_number()
                ).not_valid_before(
                    datetime.utcnow()
                ).not_valid_after(
                    datetime.utcnow() + timedelta(days=days)
                ).add_extension(
                    x509.SubjectAlternativeName([
                        x509.DNSName(hostname),
                        x509.DNSName("localhost"),
                        x509.IPAddress(ipaddress.IPv4Address("127.0.0.1")),
                    ]),
                    critical=False,
                ).sign(private_key, hashes.SHA256())
                
                # Save private key
                with open(key_file, "wb") as f:
                    f.write(private_key.private_bytes(
                        encoding=serialization.Encoding.PEM,
                        format=serialization.PrivateFormat.PKCS8,
                        encryption_algorithm=serialization.NoEncryption()
                    ))
                
                # Save certificate
                with open(cert_file, "wb") as f:
                    f.write(cert.public_bytes(serialization.Encoding.PEM))
                
                # Create CA bundle (same as cert for self-signed)
                with open(ca_file, "wb") as f:
                    f.write(cert.public_bytes(serialization.Encoding.PEM))
                
                # Set file permissions (Windows compatible)
                try:
                    os.chmod(key_file, 0o600)
                    os.chmod(cert_file, 0o644)
                    os.chmod(ca_file, 0o644)
                except (OSError, NotImplementedError):
                    pass
                
                self.log_to_widget(self.cert_log, "SUCCESS: SSL certificate generated successfully!")
                self.log_to_widget(self.cert_log, f"Certificates saved to: {output_dir.absolute()}")
                self.log_to_widget(self.cert_log, f"Private key: {key_file.name}")
                self.log_to_widget(self.cert_log, f"Certificate: {cert_file.name}")
                self.log_to_widget(self.cert_log, f"CA bundle: {ca_file.name}")
                self.log_to_widget(self.cert_log, f"Valid for {days} days")
                
                self.status_var.set("SSL certificate generated successfully")
                
            except Exception as e:
                self.log_to_widget(self.cert_log, f"ERROR: {str(e)}")
                self.status_var.set("Certificate generation failed")
        
        threading.Thread(target=worker, daemon=True).start()

    def generate_jwt_keys(self):
        """Generate JWT key pair in background thread"""
        def worker():
            try:
                self.status_var.set("Generating JWT keys...")
                self.log_to_widget(self.jwt_log, "Starting JWT key pair generation...")
                
                output_dir = Path(self.jwt_output_var.get())
                key_size = int(self.jwt_keysize_var.get())
                force = self.jwt_force_var.get()
                
                # Create output directory
                output_dir.mkdir(parents=True, exist_ok=True)
                
                # Check for existing files
                private_file = output_dir / "controller_private.pem"
                public_file = output_dir / "controller_public.pem"
                
                if not force and any(f.exists() for f in [private_file, public_file]):
                    if not messagebox.askyesno("Files Exist", 
                                             "JWT key files already exist. Overwrite?"):
                        self.status_var.set("JWT key generation cancelled")
                        return
                
                self.log_to_widget(self.jwt_log, f"Generating {key_size}-bit RSA key pair...")
                
                # Generate private key
                private_key = rsa.generate_private_key(
                    public_exponent=65537,
                    key_size=key_size,
                )
                
                # Get public key
                public_key = private_key.public_key()
                
                # Save private key
                with open(private_file, "wb") as f:
                    f.write(private_key.private_bytes(
                        encoding=serialization.Encoding.PEM,
                        format=serialization.PrivateFormat.PKCS8,
                        encryption_algorithm=serialization.NoEncryption()
                    ))
                
                # Save public key
                with open(public_file, "wb") as f:
                    f.write(public_key.public_bytes(
                        encoding=serialization.Encoding.PEM,
                        format=serialization.PublicFormat.SubjectPublicKeyInfo
                    ))
                
                # Set file permissions (Windows compatible)
                try:
                    os.chmod(private_file, 0o600)
                    os.chmod(public_file, 0o644)
                except (OSError, NotImplementedError):
                    pass
                
                self.log_to_widget(self.jwt_log, "SUCCESS: JWT key pair generated successfully!")
                self.log_to_widget(self.jwt_log, f"Keys saved to: {output_dir.absolute()}")
                self.log_to_widget(self.jwt_log, f"Private key: {private_file.name} (keep secure!)")
                self.log_to_widget(self.jwt_log, f"Public key: {public_file.name} (distribute to clients)")
                self.log_to_widget(self.jwt_log, "")
                self.log_to_widget(self.jwt_log, "Next steps:")
                self.log_to_widget(self.jwt_log, "1. Keep the private key secure on your server")
                self.log_to_widget(self.jwt_log, "2. Copy the public key to all client machines")
                self.log_to_widget(self.jwt_log, "3. Configure server to use the private key")
                self.log_to_widget(self.jwt_log, "4. Configure clients to use the public key")
                
                self.status_var.set("JWT keys generated successfully")
                
            except Exception as e:
                self.log_to_widget(self.jwt_log, f"ERROR: {str(e)}")
                self.status_var.set("JWT key generation failed")
        
        threading.Thread(target=worker, daemon=True).start()

    def run_client_installer(self):
        """Run the NSIS client installer"""
        try:
            installer_path = Path(__file__).parent.parent / "installers-exe" / "ArcVault-Client-Installer.exe"
            
            if not installer_path.exists():
                messagebox.showerror("Installer Not Found", 
                                   f"Client installer not found at:\n{installer_path}")
                return
            
            self.log_to_widget(self.installer_log, "Launching client installer...")
            self.status_var.set("Running client installer...")
            
            # Run installer as administrator
            subprocess.Popen([
                "powershell", "-Command", 
                f"Start-Process '{installer_path}' -Verb RunAs"
            ])
            
            self.log_to_widget(self.installer_log, "Client installer launched successfully")
            self.status_var.set("Client installer launched")
            
        except Exception as e:
            self.log_to_widget(self.installer_log, f"ERROR: Failed to launch installer: {e}")
            self.status_var.set("Failed to launch installer")

    def service_action(self, action):
        """Perform service management action"""
        def worker():
            try:
                service_exe = Path("C:/Program Files (x86)/ArcVault/ArcVaultAgentService.exe")
                if not service_exe.exists():
                    service_exe = Path("C:/Program Files/ArcVault/ArcVaultAgentService.exe")
                
                if not service_exe.exists():
                    self.log_to_widget(self.installer_log, 
                                     "ERROR: ArcVault service executable not found. Install client first.")
                    return
                
                self.log_to_widget(self.installer_log, f"Performing service action: {action}")
                
                if action == "install":
                    cmd = [str(service_exe), "install"]
                elif action == "start":
                    cmd = ["net", "start", "ArcVaultAgent"]
                elif action == "stop":
                    cmd = ["net", "stop", "ArcVaultAgent"]
                elif action == "uninstall":
                    cmd = [str(service_exe), "remove"]
                
                # Run command as administrator
                result = subprocess.run(
                    ["powershell", "-Command", f"Start-Process '{' '.join(cmd)}' -Verb RunAs -Wait"],
                    capture_output=True, text=True, timeout=30
                )
                
                if result.returncode == 0:
                    self.log_to_widget(self.installer_log, f"Service {action} completed successfully")
                else:
                    self.log_to_widget(self.installer_log, f"Service {action} failed: {result.stderr}")
                
            except Exception as e:
                self.log_to_widget(self.installer_log, f"ERROR: Service {action} failed: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def check_service_status(self):
        """Check Windows service status"""
        def worker():
            try:
                self.log_to_widget(self.installer_log, "Checking service status...")
                
                # Check Windows service
                result = subprocess.run(
                    ["sc", "query", "ArcVaultAgent"],
                    capture_output=True, text=True
                )
                
                if result.returncode == 0:
                    self.log_to_widget(self.installer_log, "Service Status:")
                    for line in result.stdout.split('\n'):
                        if line.strip():
                            self.log_to_widget(self.installer_log, f"  {line.strip()}")
                else:
                    self.log_to_widget(self.installer_log, "Service not found or not installed")
                
                # Test agent health
                self.test_agent_connection()
                
            except Exception as e:
                self.log_to_widget(self.installer_log, f"ERROR: Failed to check service status: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def test_agent_connection(self):
        """Test agent HTTP connection"""
        def worker():
            try:
                import urllib.request
                import json
                
                self.log_to_widget(self.installer_log, "Testing agent connection...")
                
                # Test health endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/health", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_to_widget(self.installer_log, f"Agent Health: {data}")
                except Exception as e:
                    self.log_to_widget(self.installer_log, f"Agent health check failed: {e}")
                
                # Test info endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/info", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_to_widget(self.installer_log, f"Agent Info: {data}")
                except Exception as e:
                    self.log_to_widget(self.installer_log, f"Agent info check failed: {e}")
                
            except Exception as e:
                self.log_to_widget(self.installer_log, f"ERROR: Connection test failed: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def configure_agent(self):
        """Configure agent with server URL and token"""
        server_url = self.server_url_var.get().strip()
        auth_token = self.auth_token_var.get().strip()
        
        if not server_url:
            messagebox.showerror("Missing Information", "Server URL is required")
            return
        
        if not auth_token:
            messagebox.showerror("Missing Information", "Authentication token is required")
            return
        
        self.log_to_widget(self.installer_log, f"Configuring agent for server: {server_url}")
        
        # This would typically write to a config file or registry
        # For now, just log the configuration
        self.log_to_widget(self.installer_log, "Configuration saved (placeholder)")
        self.log_to_widget(self.installer_log, "Note: Manual configuration may be required")


def main():
    """Main entry point"""
    root = tk.Tk()
    app = ArcVaultGUI(root)
    
    # Center window on screen
    root.update_idletasks()
    x = (root.winfo_screenwidth() // 2) - (root.winfo_width() // 2)
    y = (root.winfo_screenheight() // 2) - (root.winfo_height() // 2)
    root.geometry(f"+{x}+{y}")
    
    root.mainloop()


if __name__ == "__main__":
    main()
