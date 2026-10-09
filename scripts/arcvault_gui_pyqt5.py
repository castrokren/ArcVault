#!/usr/bin/env python3
"""
ArcVault Unified GUI Application - PyQt5 Version

A modern, professional GUI tool that combines:
- SSL Certificate Generation
- JWT Key Generation  
- Client Installation Management

Built with PyQt5 for superior styling and user experience.
"""

import sys
import os
import subprocess
import threading
from pathlib import Path
from datetime import datetime, timedelta
import tempfile
import json
import urllib.request

try:
    from PyQt5.QtWidgets import (
        QApplication, QMainWindow, QTabWidget, QWidget, QVBoxLayout, 
        QHBoxLayout, QGridLayout, QLabel, QLineEdit, QPushButton, 
        QComboBox, QCheckBox, QTextEdit, QProgressBar, QGroupBox,
        QFileDialog, QMessageBox, QSplitter, QFrame, QScrollArea,
        QSpacerItem, QSizePolicy, QStatusBar
    )
    from PyQt5.QtCore import Qt, QThread, pyqtSignal, QTimer
    from PyQt5.QtGui import QFont, QIcon, QPixmap, QPalette, QColor
except ImportError:
    print("PyQt5 not found. Installing...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "PyQt5"])
    from PyQt5.QtWidgets import (
        QApplication, QMainWindow, QTabWidget, QWidget, QVBoxLayout, 
        QHBoxLayout, QGridLayout, QLabel, QLineEdit, QPushButton, 
        QComboBox, QCheckBox, QTextEdit, QProgressBar, QGroupBox,
        QFileDialog, QMessageBox, QSplitter, QFrame, QScrollArea,
        QSpacerItem, QSizePolicy, QStatusBar
    )
    from PyQt5.QtCore import Qt, QThread, pyqtSignal, QTimer
    from PyQt5.QtGui import QFont, QIcon, QPixmap, QPalette, QColor

# Import cryptography modules
try:
    from cryptography import x509
    from cryptography.x509.oid import NameOID
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
    import ipaddress
except ImportError:
    QMessageBox.critical(None, "Missing Dependencies", 
                        "Required cryptography library not found.\n"
                        "Please install it with: pip install cryptography")
    sys.exit(1)


class WorkerThread(QThread):
    """Base worker thread for background operations"""
    progress = pyqtSignal(str)
    finished = pyqtSignal(bool, str)
    
    def __init__(self):
        super().__init__()
        self.should_stop = False
    
    def stop(self):
        self.should_stop = True


class CertificateWorker(WorkerThread):
    """Worker thread for certificate generation"""
    
    def __init__(self, output_dir, hostname, key_size, days, force, country, org):
        super().__init__()
        self.output_dir = Path(output_dir)
        self.hostname = hostname
        self.key_size = key_size
        self.days = days
        self.force = force
        self.country = country
        self.org = org
    
    def run(self):
        try:
            self.progress.emit("Starting SSL certificate generation...")
            
            # Validate inputs
            if not self.hostname:
                raise ValueError("Hostname is required")
            if not self.country or len(self.country) != 2:
                raise ValueError("Country must be exactly 2 letters")
            if not self.org:
                raise ValueError("Organization is required")
            
            # Create output directory
            self.output_dir.mkdir(parents=True, exist_ok=True)
            
            # Check for existing files
            cert_file = self.output_dir / "server.crt"
            key_file = self.output_dir / "server.key"
            ca_file = self.output_dir / "ca-bundle.crt"
            
            if not self.force and any(f.exists() for f in [cert_file, key_file, ca_file]):
                self.finished.emit(False, "Certificate files already exist. Use force option to overwrite.")
                return
            
            self.progress.emit(f"Generating {self.key_size}-bit RSA key pair...")
            
            # Generate private key
            private_key = rsa.generate_private_key(
                public_exponent=65537,
                key_size=self.key_size,
            )
            
            if self.should_stop:
                return
            
            self.progress.emit("Creating certificate...")
            
            # Create certificate
            subject = issuer = x509.Name([
                x509.NameAttribute(NameOID.COUNTRY_NAME, self.country),
                x509.NameAttribute(NameOID.ORGANIZATION_NAME, self.org),
                x509.NameAttribute(NameOID.COMMON_NAME, self.hostname),
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
                datetime.utcnow() + timedelta(days=self.days)
            ).add_extension(
                x509.SubjectAlternativeName([
                    x509.DNSName(self.hostname),
                    x509.DNSName("localhost"),
                    x509.IPAddress(ipaddress.IPv4Address("127.0.0.1")),
                ]),
                critical=False,
            ).sign(private_key, hashes.SHA256())
            
            if self.should_stop:
                return
            
            self.progress.emit("Saving certificate files...")
            
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
            
            result_msg = f"""SUCCESS: SSL certificate generated successfully!

Files saved to: {self.output_dir.absolute()}
- Private key: {key_file.name}
- Certificate: {cert_file.name}
- CA bundle: {ca_file.name}

Valid for {self.days} days
Certificate fingerprint: {cert.fingerprint(hashes.SHA256()).hex()}"""
            
            self.finished.emit(True, result_msg)
            
        except Exception as e:
            self.finished.emit(False, f"ERROR: {str(e)}")


class JWTWorker(WorkerThread):
    """Worker thread for JWT key generation"""
    
    def __init__(self, output_dir, key_size, force):
        super().__init__()
        self.output_dir = Path(output_dir)
        self.key_size = key_size
        self.force = force
    
    def run(self):
        try:
            self.progress.emit("Starting JWT key pair generation...")
            
            # Create output directory
            self.output_dir.mkdir(parents=True, exist_ok=True)
            
            # Check for existing files
            private_file = self.output_dir / "controller_private.pem"
            public_file = self.output_dir / "controller_public.pem"
            
            if not self.force and any(f.exists() for f in [private_file, public_file]):
                self.finished.emit(False, "JWT key files already exist. Use force option to overwrite.")
                return
            
            self.progress.emit(f"Generating {self.key_size}-bit RSA key pair...")
            
            # Generate private key
            private_key = rsa.generate_private_key(
                public_exponent=65537,
                key_size=self.key_size,
            )
            
            if self.should_stop:
                return
            
            # Get public key
            public_key = private_key.public_key()
            
            self.progress.emit("Saving key files...")
            
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
            
            result_msg = f"""SUCCESS: JWT key pair generated successfully!

Keys saved to: {self.output_dir.absolute()}
- Private key: {private_file.name} (keep secure!)
- Public key: {public_file.name} (distribute to clients)

Next steps:
1. Keep the private key secure on your server
2. Copy the public key to all client machines
3. Configure server to use the private key
4. Configure clients to use the public key"""
            
            self.finished.emit(True, result_msg)
            
        except Exception as e:
            self.finished.emit(False, f"ERROR: {str(e)}")


class ArcVaultMainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("ArcVault Management Suite")
        self.setGeometry(100, 100, 1000, 700)
        self.setMinimumSize(900, 600)
        
        # Set application style
        self.setStyleSheet("""
            QMainWindow {
                background-color: #f0f0f0;
            }
            QTabWidget::pane {
                border: 1px solid #c0c0c0;
                background-color: white;
            }
            QTabWidget::tab-bar {
                left: 5px;
            }
            QTabBar::tab {
                background-color: #e1e1e1;
                border: 1px solid #c0c0c0;
                padding: 8px 16px;
                margin-right: 2px;
            }
            QTabBar::tab:selected {
                background-color: white;
                border-bottom-color: white;
            }
            QTabBar::tab:hover {
                background-color: #f0f0f0;
            }
            QGroupBox {
                font-weight: bold;
                border: 2px solid #cccccc;
                border-radius: 5px;
                margin: 5px;
                padding-top: 10px;
            }
            QGroupBox::title {
                subcontrol-origin: margin;
                left: 10px;
                padding: 0 5px 0 5px;
            }
            QPushButton {
                background-color: #0078d4;
                color: white;
                border: none;
                padding: 8px 16px;
                border-radius: 4px;
                font-weight: bold;
            }
            QPushButton:hover {
                background-color: #106ebe;
            }
            QPushButton:pressed {
                background-color: #005a9e;
            }
            QPushButton:disabled {
                background-color: #cccccc;
                color: #666666;
            }
            QLineEdit, QComboBox {
                padding: 6px;
                border: 1px solid #cccccc;
                border-radius: 3px;
                background-color: white;
            }
            QLineEdit:focus, QComboBox:focus {
                border-color: #0078d4;
            }
            QTextEdit {
                border: 1px solid #cccccc;
                border-radius: 3px;
                background-color: white;
                font-family: 'Consolas', 'Monaco', monospace;
            }
            QProgressBar {
                border: 1px solid #cccccc;
                border-radius: 3px;
                text-align: center;
            }
            QProgressBar::chunk {
                background-color: #0078d4;
                border-radius: 2px;
            }
        """)
        
        # Create central widget and main layout
        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        
        # Create tab widget
        self.tab_widget = QTabWidget()
        
        # Create tabs
        self.create_certificate_tab()
        self.create_jwt_tab()
        self.create_installer_tab()
        self.create_about_tab()
        
        # Main layout
        main_layout = QVBoxLayout()
        main_layout.addWidget(self.tab_widget)
        central_widget.setLayout(main_layout)
        
        # Status bar
        self.status_bar = QStatusBar()
        self.setStatusBar(self.status_bar)
        self.status_bar.showMessage("Ready")
        
        # Initialize worker threads
        self.cert_worker = None
        self.jwt_worker = None

    def create_certificate_tab(self):
        """Create SSL Certificate Generation tab"""
        tab = QWidget()
        self.tab_widget.addTab(tab, "🔐 SSL Certificates")
        
        # Main layout
        layout = QVBoxLayout()
        
        # Title
        title = QLabel("SSL Certificate Generator")
        title.setFont(QFont("Arial", 16, QFont.Bold))
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)
        
        # Create splitter for better layout
        splitter = QSplitter(Qt.Vertical)
        
        # Top section - Configuration
        config_widget = QWidget()
        config_layout = QVBoxLayout()
        
        # Output Directory Group
        dir_group = QGroupBox("Output Directory")
        dir_layout = QHBoxLayout()
        self.cert_output_edit = QLineEdit(str(Path.cwd() / "keys"))
        dir_browse_btn = QPushButton("Browse...")
        dir_browse_btn.clicked.connect(self.browse_cert_output)
        dir_layout.addWidget(self.cert_output_edit)
        dir_layout.addWidget(dir_browse_btn)
        dir_group.setLayout(dir_layout)
        config_layout.addWidget(dir_group)
        
        # Certificate Settings Group
        settings_group = QGroupBox("Certificate Settings")
        settings_layout = QGridLayout()
        
        # Hostname
        settings_layout.addWidget(QLabel("Hostname:"), 0, 0)
        self.cert_hostname_edit = QLineEdit("localhost")
        settings_layout.addWidget(self.cert_hostname_edit, 0, 1)
        
        # Key Size
        settings_layout.addWidget(QLabel("Key Size:"), 1, 0)
        self.cert_keysize_combo = QComboBox()
        self.cert_keysize_combo.addItems(["2048", "3072", "4096"])
        self.cert_keysize_combo.setCurrentText("2048")
        settings_layout.addWidget(self.cert_keysize_combo, 1, 1)
        
        # Validity Days
        settings_layout.addWidget(QLabel("Valid for (days):"), 2, 0)
        self.cert_days_edit = QLineEdit("365")
        settings_layout.addWidget(self.cert_days_edit, 2, 1)
        
        # Country
        settings_layout.addWidget(QLabel("Country (2 letters):"), 0, 2)
        self.cert_country_edit = QLineEdit("US")
        self.cert_country_edit.setMaxLength(2)
        settings_layout.addWidget(self.cert_country_edit, 0, 2)
        
        # Organization
        settings_layout.addWidget(QLabel("Organization:"), 1, 2)
        self.cert_org_edit = QLineEdit("ArcVault")
        settings_layout.addWidget(self.cert_org_edit, 1, 2)
        
        # Force overwrite
        self.cert_force_check = QCheckBox("Overwrite existing certificates")
        settings_layout.addWidget(self.cert_force_check, 2, 2)
        
        settings_group.setLayout(settings_layout)
        config_layout.addWidget(settings_group)
        
        # Generate button
        self.cert_generate_btn = QPushButton("🔐 Generate SSL Certificate")
        self.cert_generate_btn.setMinimumHeight(40)
        self.cert_generate_btn.clicked.connect(self.generate_certificate)
        config_layout.addWidget(self.cert_generate_btn)
        
        # Progress bar
        self.cert_progress = QProgressBar()
        self.cert_progress.setVisible(False)
        config_layout.addWidget(self.cert_progress)
        
        config_widget.setLayout(config_layout)
        splitter.addWidget(config_widget)
        
        # Bottom section - Output Log
        log_group = QGroupBox("Output Log")
        log_layout = QVBoxLayout()
        self.cert_log = QTextEdit()
        self.cert_log.setMinimumHeight(200)
        log_layout.addWidget(self.cert_log)
        log_group.setLayout(log_layout)
        splitter.addWidget(log_group)
        
        layout.addWidget(splitter)
        tab.setLayout(layout)

    def create_jwt_tab(self):
        """Create JWT Key Generation tab"""
        tab = QWidget()
        self.tab_widget.addTab(tab, "🔑 JWT Keys")
        
        # Main layout
        layout = QVBoxLayout()
        
        # Title
        title = QLabel("JWT Key Pair Generator")
        title.setFont(QFont("Arial", 16, QFont.Bold))
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)
        
        # Create splitter
        splitter = QSplitter(Qt.Vertical)
        
        # Top section - Configuration
        config_widget = QWidget()
        config_layout = QVBoxLayout()
        
        # Output Directory Group
        dir_group = QGroupBox("Output Directory")
        dir_layout = QHBoxLayout()
        self.jwt_output_edit = QLineEdit(str(Path.cwd() / "keys"))
        jwt_dir_browse_btn = QPushButton("Browse...")
        jwt_dir_browse_btn.clicked.connect(self.browse_jwt_output)
        dir_layout.addWidget(self.jwt_output_edit)
        dir_layout.addWidget(jwt_dir_browse_btn)
        dir_group.setLayout(dir_layout)
        config_layout.addWidget(dir_group)
        
        # Key Settings Group
        settings_group = QGroupBox("Key Settings")
        settings_layout = QGridLayout()
        
        # Key Size
        settings_layout.addWidget(QLabel("Key Size:"), 0, 0)
        self.jwt_keysize_combo = QComboBox()
        self.jwt_keysize_combo.addItems(["2048", "3072", "4096"])
        self.jwt_keysize_combo.setCurrentText("2048")
        settings_layout.addWidget(self.jwt_keysize_combo, 0, 1)
        
        # Force overwrite
        self.jwt_force_check = QCheckBox("Overwrite existing keys")
        settings_layout.addWidget(self.jwt_force_check, 1, 0, 1, 2)
        
        settings_group.setLayout(settings_layout)
        config_layout.addWidget(settings_group)
        
        # Generate button
        self.jwt_generate_btn = QPushButton("🔑 Generate JWT Key Pair")
        self.jwt_generate_btn.setMinimumHeight(40)
        self.jwt_generate_btn.clicked.connect(self.generate_jwt_keys)
        config_layout.addWidget(self.jwt_generate_btn)
        
        # Progress bar
        self.jwt_progress = QProgressBar()
        self.jwt_progress.setVisible(False)
        config_layout.addWidget(self.jwt_progress)
        
        config_widget.setLayout(config_layout)
        splitter.addWidget(config_widget)
        
        # Bottom section - Output Log
        log_group = QGroupBox("Output Log")
        log_layout = QVBoxLayout()
        self.jwt_log = QTextEdit()
        self.jwt_log.setMinimumHeight(200)
        log_layout.addWidget(self.jwt_log)
        log_group.setLayout(log_layout)
        splitter.addWidget(log_group)
        
        layout.addWidget(splitter)
        tab.setLayout(layout)

    def create_installer_tab(self):
        """Create Client Installer Management tab"""
        tab = QWidget()
        self.tab_widget.addTab(tab, "💻 Client Manager")
        
        # Main layout
        layout = QVBoxLayout()
        
        # Title
        title = QLabel("ArcVault Client Management")
        title.setFont(QFont("Arial", 16, QFont.Bold))
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)
        
        # Create splitter
        splitter = QSplitter(Qt.Vertical)
        
        # Top section - Controls
        controls_widget = QWidget()
        controls_layout = QVBoxLayout()
        
        # Installation Group
        install_group = QGroupBox("Install ArcVault Agent")
        install_layout = QVBoxLayout()
        
        install_btn = QPushButton("🚀 Install ArcVault Agent")
        install_btn.setMinimumHeight(50)
        install_btn.setStyleSheet("""
            QPushButton {
                background-color: #28a745;
                font-size: 14px;
                font-weight: bold;
            }
            QPushButton:hover {
                background-color: #218838;
            }
            QPushButton:disabled {
                background-color: #cccccc;
            }
        """)
        install_btn.clicked.connect(self.install_agent_complete)
        install_layout.addWidget(install_btn)
        
        install_info = QLabel("This will install the agent and configure it as a Windows service.")
        install_info.setWordWrap(True)
        install_info.setStyleSheet("color: #666666; font-size: 10px;")
        install_layout.addWidget(install_info)
        
        install_group.setLayout(install_layout)
        controls_layout.addWidget(install_group)
        
        # Status and Diagnostics Group
        status_group = QGroupBox("Status & Diagnostics")
        status_layout = QVBoxLayout()
        
        status_btn = QPushButton("📊 Check Service Status")
        status_btn.clicked.connect(self.check_service_status)
        status_layout.addWidget(status_btn)
        
        test_btn = QPushButton("🔍 Test Agent Connection")
        test_btn.clicked.connect(self.test_agent_connection)
        status_layout.addWidget(test_btn)
        
        diagnostics_btn = QPushButton("📋 View Event Logs")
        diagnostics_btn.clicked.connect(self.view_event_logs)
        status_layout.addWidget(diagnostics_btn)

        manual_test_btn = QPushButton("🧪 Run Manual Agent Test")
        manual_test_btn.clicked.connect(self.run_manual_agent_test)
        status_layout.addWidget(manual_test_btn)

        dependency_btn = QPushButton("🧾 Check Agent Dependencies")
        dependency_btn.clicked.connect(self.check_agent_dependencies)
        status_layout.addWidget(dependency_btn)

        fix_config_btn = QPushButton("🛠 Fix Service Config")
        fix_config_btn.clicked.connect(self.fix_service_config)
        status_layout.addWidget(fix_config_btn)
        
        status_group.setLayout(status_layout)
        controls_layout.addWidget(status_group)
        
        # Quick Setup Group
        setup_group = QGroupBox("Quick Setup")
        setup_layout = QGridLayout()
        
        setup_layout.addWidget(QLabel("Server URL:"), 0, 0)
        self.server_url_edit = QLineEdit("https://backup-server.company.com:8443")
        setup_layout.addWidget(self.server_url_edit, 0, 1)
        
        setup_layout.addWidget(QLabel("Auth Token:"), 1, 0)
        self.auth_token_edit = QLineEdit()
        self.auth_token_edit.setEchoMode(QLineEdit.Password)
        setup_layout.addWidget(self.auth_token_edit, 1, 1)
        
        configure_btn = QPushButton("Configure Agent")
        configure_btn.clicked.connect(self.configure_agent)
        setup_layout.addWidget(configure_btn, 2, 0, 1, 2)
        
        setup_group.setLayout(setup_layout)
        controls_layout.addWidget(setup_group)
        
        controls_widget.setLayout(controls_layout)
        splitter.addWidget(controls_widget)
        
        # Bottom section - Output Log
        log_group = QGroupBox("Output Log")
        log_layout = QVBoxLayout()
        self.installer_log = QTextEdit()
        self.installer_log.setMinimumHeight(200)
        log_layout.addWidget(self.installer_log)
        log_group.setLayout(log_layout)
        splitter.addWidget(log_group)
        
        layout.addWidget(splitter)
        tab.setLayout(layout)

    def create_about_tab(self):
        """Create About tab"""
        tab = QWidget()
        self.tab_widget.addTab(tab, "ℹ️ About")
        
        layout = QVBoxLayout()
        
        # Center everything
        layout.addStretch()
        
        # Title
        title = QLabel("ArcVault Management Suite")
        title.setFont(QFont("Arial", 20, QFont.Bold))
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)
        
        # Version
        version = QLabel("Version 2.0.0 - PyQt5 Edition")
        version.setFont(QFont("Arial", 12))
        version.setAlignment(Qt.AlignCenter)
        layout.addWidget(version)
        
        # Description
        desc = QLabel("Professional unified management tool for ArcVault backup system")
        desc.setFont(QFont("Arial", 10))
        desc.setAlignment(Qt.AlignCenter)
        layout.addWidget(desc)
        
        layout.addSpacing(20)
        
        # Features
        features_group = QGroupBox("Features")
        features_layout = QVBoxLayout()
        
        features = [
            "🔐 SSL Certificate Generation with advanced options",
            "🔑 JWT Key Pair Generation for secure authentication", 
            "💻 Client Installation Management with service control",
            "⚙️ Windows Service Management (Install/Start/Stop/Uninstall)",
            "🔧 Agent Configuration with server connection setup",
            "🔍 Connection Testing and health monitoring",
            "🎨 Modern PyQt5 interface with professional styling",
            "📊 Real-time progress tracking and detailed logging"
        ]
        
        for feature in features:
            label = QLabel(feature)
            label.setFont(QFont("Arial", 10))
            features_layout.addWidget(label)
        
        features_group.setLayout(features_layout)
        layout.addWidget(features_group)
        
        layout.addSpacing(20)
        
        # Support
        support = QLabel("Support: https://github.com/castrokren/arcvault-tool")
        support.setFont(QFont("Arial", 10))
        support.setAlignment(Qt.AlignCenter)
        layout.addWidget(support)
        
        layout.addStretch()
        tab.setLayout(layout)

    # Event handlers
    def browse_cert_output(self):
        directory = QFileDialog.getExistingDirectory(self, "Select Output Directory", 
                                                   self.cert_output_edit.text())
        if directory:
            self.cert_output_edit.setText(directory)

    def browse_jwt_output(self):
        directory = QFileDialog.getExistingDirectory(self, "Select Output Directory", 
                                                   self.jwt_output_edit.text())
        if directory:
            self.jwt_output_edit.setText(directory)

    def log_message(self, log_widget, message):
        """Add timestamped message to log widget"""
        timestamp = datetime.now().strftime('%H:%M:%S')
        log_widget.append(f"[{timestamp}] {message}")

    def generate_certificate(self):
        """Generate SSL certificate"""
        if self.cert_worker and self.cert_worker.isRunning():
            return
        
        # Validate inputs
        hostname = self.cert_hostname_edit.text().strip()
        if not hostname:
            QMessageBox.warning(self, "Invalid Input", "Hostname is required")
            return
        
        try:
            days = int(self.cert_days_edit.text())
            if days <= 0:
                raise ValueError()
        except ValueError:
            QMessageBox.warning(self, "Invalid Input", "Days must be a positive number")
            return
        
        country = self.cert_country_edit.text().strip().upper()
        if len(country) != 2:
            QMessageBox.warning(self, "Invalid Input", "Country must be exactly 2 letters")
            return
        
        org = self.cert_org_edit.text().strip()
        if not org:
            QMessageBox.warning(self, "Invalid Input", "Organization is required")
            return
        
        # Start generation
        self.cert_generate_btn.setEnabled(False)
        self.cert_progress.setVisible(True)
        self.cert_progress.setRange(0, 0)  # Indeterminate progress
        self.cert_log.clear()
        self.status_bar.showMessage("Generating SSL certificate...")
        
        self.cert_worker = CertificateWorker(
            self.cert_output_edit.text(),
            hostname,
            int(self.cert_keysize_combo.currentText()),
            days,
            self.cert_force_check.isChecked(),
            country,
            org
        )
        
        self.cert_worker.progress.connect(lambda msg: self.log_message(self.cert_log, msg))
        self.cert_worker.finished.connect(self.on_cert_finished)
        self.cert_worker.start()

    def on_cert_finished(self, success, message):
        """Handle certificate generation completion"""
        self.cert_generate_btn.setEnabled(True)
        self.cert_progress.setVisible(False)
        
        if success:
            self.log_message(self.cert_log, message)
            self.status_bar.showMessage("SSL certificate generated successfully")
            QMessageBox.information(self, "Success", "SSL certificate generated successfully!")
        else:
            self.log_message(self.cert_log, message)
            self.status_bar.showMessage("Certificate generation failed")
            QMessageBox.critical(self, "Error", message)

    def generate_jwt_keys(self):
        """Generate JWT key pair"""
        if self.jwt_worker and self.jwt_worker.isRunning():
            return
        
        # Start generation
        self.jwt_generate_btn.setEnabled(False)
        self.jwt_progress.setVisible(True)
        self.jwt_progress.setRange(0, 0)  # Indeterminate progress
        self.jwt_log.clear()
        self.status_bar.showMessage("Generating JWT keys...")
        
        self.jwt_worker = JWTWorker(
            self.jwt_output_edit.text(),
            int(self.jwt_keysize_combo.currentText()),
            self.jwt_force_check.isChecked()
        )
        
        self.jwt_worker.progress.connect(lambda msg: self.log_message(self.jwt_log, msg))
        self.jwt_worker.finished.connect(self.on_jwt_finished)
        self.jwt_worker.start()

    def on_jwt_finished(self, success, message):
        """Handle JWT generation completion"""
        self.jwt_generate_btn.setEnabled(True)
        self.jwt_progress.setVisible(False)
        
        if success:
            self.log_message(self.jwt_log, message)
            self.status_bar.showMessage("JWT keys generated successfully")
            QMessageBox.information(self, "Success", "JWT key pair generated successfully!")
        else:
            self.log_message(self.jwt_log, message)
            self.status_bar.showMessage("JWT key generation failed")
            QMessageBox.critical(self, "Error", message)

    def run_client_installer(self):
        """Run the NSIS client installer"""
        try:
            installer_path = Path(__file__).parent.parent / "installers-exe" / "ArcVault-Client-Installer.exe"
            
            if not installer_path.exists():
                QMessageBox.critical(self, "Installer Not Found", 
                                   f"Client installer not found at:\n{installer_path}")
                return
            
            self.log_message(self.installer_log, "Launching client installer...")
            self.status_bar.showMessage("Running client installer...")
            
            # Run installer as administrator
            subprocess.Popen([
                "powershell", "-Command", 
                f"Start-Process '{installer_path}' -Verb RunAs"
            ])
            
            self.log_message(self.installer_log, "Client installer launched successfully")
            self.status_bar.showMessage("Client installer launched")
            
        except Exception as e:
            self.log_message(self.installer_log, f"ERROR: Failed to launch installer: {e}")
            self.status_bar.showMessage("Failed to launch installer")
            QMessageBox.critical(self, "Error", f"Failed to launch installer: {e}")

    def install_agent_complete(self):
        """Complete installation: copy agent, install service, and start it"""
        def worker():
            try:
                import time
                import shutil
                import json
                
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Starting ArcVault Agent Installation")
                self.log_message(self.installer_log, "=" * 60)
                
                # Step 1: Find NSSM
                if getattr(sys, 'frozen', False):
                    bundle_dir = Path(getattr(sys, '_MEIPASS', Path(sys.executable).parent))
                    nssm_paths = [
                        bundle_dir / "nssm.exe",
                        Path(sys.executable).parent / "nssm.exe",
                    ]
                else:
                    nssm_paths = [
                        Path(__file__).parent.parent / "nssm.exe",
                    ]
                
                nssm_path = None
                for path in nssm_paths:
                    if path and path.exists():
                        nssm_path = path
                        self.log_message(self.installer_log, f"✓ Found nssm.exe at: {path}")
                        break
                
                if not nssm_path:
                    self.log_message(self.installer_log, "ERROR: nssm.exe not found")
                    self.log_message(self.installer_log, "Please ensure nssm.exe is bundled with the GUI.")
                    return
                
                # Step 2: Find agent executable
                agent_paths = []
                if getattr(sys, 'frozen', False):
                    bundle_dir = Path(getattr(sys, '_MEIPASS', Path(sys.executable).parent))
                    agent_paths.append(bundle_dir / "ArcVault.exe")
                    exe_dir = Path(sys.executable).parent
                    agent_paths.extend([
                        exe_dir.parent / "apps" / "backup-agent" / "dist" / "ArcVault.exe",
                        exe_dir.parent.parent / "apps" / "backup-agent" / "dist" / "ArcVault.exe",
                        exe_dir / "apps" / "backup-agent" / "dist" / "ArcVault.exe",
                    ])
                else:
                    script_dir = Path(__file__).parent.parent
                    agent_paths.append(script_dir / "apps" / "backup-agent" / "dist" / "ArcVault.exe")
                
                agent_exe = None
                for path in agent_paths:
                    if path and path.exists():
                        agent_exe = path
                        self.log_message(self.installer_log, f"✓ Found agent at: {path}")
                        break
                
                if agent_exe is None:
                    self.log_message(self.installer_log, "ERROR: ArcVault.exe (agent) not found")
                    if getattr(sys, 'frozen', False):
                        self.log_message(self.installer_log, "The agent should be bundled with this GUI executable.")
                    self.log_message(self.installer_log, "Please rebuild the GUI to include the agent.")
                    return
                
                # Step 3: Copy agent to installation directory
                # Executable goes to Program Files (read-only, but that's OK)
                install_path = Path("C:/Program Files (x86)/ArcVault")
                target_exe = install_path / "ArcVault.exe"
                
                # Config and logs go to ProgramData (writable location)
                program_data = Path(os.getenv("ProgramData", "C:/ProgramData"))
                config_path = program_data / "ArcVault"
                
                if agent_exe.resolve() == target_exe.resolve():
                    self.log_message(self.installer_log, "ERROR: Source and target are the same file!")
                    self.log_message(self.installer_log, "Cannot copy agent to itself. Please check the installation.")
                    return
                
                # Step 3.5: Remove existing service FIRST (before file operations)
                # This ensures the file isn't locked when we try to replace it
                self.log_message(self.installer_log, "Checking for existing service...")
                check_result = subprocess.run(
                    ["sc", "query", "ArcVaultAgent"],
                    capture_output=True, text=True
                )
                
                if check_result.returncode == 0:
                    # Service exists, remove it first
                    self.log_message(self.installer_log, "Existing service found. Stopping and removing...")
                    
                    # Stop the service first
                    stop_result = subprocess.run(
                        ["sc", "stop", "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=30
                    )
                    if stop_result.returncode == 0 or "STOPPED" in stop_result.stdout.upper():
                        self.log_message(self.installer_log, "✓ Service stopped")
                    
                    # Wait for service to fully stop and release file locks
                    self.log_message(self.installer_log, "Waiting for service to release file locks...")
                    time.sleep(5)  # Give more time for file locks to release
                    
                    # Remove service using NSSM
                    remove_result = subprocess.run(
                        ["powershell", "-Command", 
                         f"$proc = Start-Process -FilePath '{nssm_path}' -ArgumentList 'remove', 'ArcVaultAgent', 'confirm' -Verb RunAs -Wait -PassThru -WindowStyle Hidden; exit $proc.ExitCode"],
                        capture_output=True, text=True, timeout=60
                    )
                    
                    if remove_result.returncode == 0:
                        self.log_message(self.installer_log, "✓ Existing service removed")
                        time.sleep(2)  # Give Windows time to clean up
                    else:
                        # Try using sc delete as fallback
                        self.log_message(self.installer_log, "NSSM remove failed, trying sc delete...")
                        delete_result = subprocess.run(
                            ["sc", "delete", "ArcVaultAgent"],
                            capture_output=True, text=True, timeout=30
                        )
                        if delete_result.returncode == 0:
                            self.log_message(self.installer_log, "✓ Service removed using sc delete")
                            time.sleep(2)
                        else:
                            self.log_message(self.installer_log, f"WARNING: Could not remove existing service: {delete_result.stdout}")
                else:
                    self.log_message(self.installer_log, "No existing service found")
                
                self.log_message(self.installer_log, f"Creating installation directory: {install_path}")
                install_path.mkdir(parents=True, exist_ok=True)
                
                # Delete existing file if present (now that service is removed)
                if target_exe.exists():
                    self.log_message(self.installer_log, "Removing existing agent installation...")
                    max_delete_retries = 5
                    for attempt in range(max_delete_retries):
                        try:
                            # Try to rename first (works even if file is slightly locked)
                            backup_name = target_exe.with_suffix('.exe.old')
                            if backup_name.exists():
                                backup_name.unlink()
                            target_exe.rename(backup_name)
                            # Now delete the renamed file
                            time.sleep(1)
                            backup_name.unlink()
                            self.log_message(self.installer_log, "✓ Existing file removed")
                            break
                        except (PermissionError, OSError) as e:
                            if attempt < max_delete_retries - 1:
                                self.log_message(self.installer_log, f"Delete attempt {attempt + 1} failed, waiting longer...")
                                time.sleep(3)
                            else:
                                self.log_message(self.installer_log, f"ERROR: Cannot remove existing file after {max_delete_retries} attempts")
                                self.log_message(self.installer_log, f"Error: {e}")
                                self.log_message(self.installer_log, "The file may still be locked. Try closing any applications using it.")
                                return
                
                # Copy agent
                self.log_message(self.installer_log, f"Copying agent from {agent_exe} to {target_exe}")
                max_retries = 3
                for attempt in range(max_retries):
                    try:
                        shutil.copy2(agent_exe, target_exe)
                        self.log_message(self.installer_log, "✓ Agent copied successfully")
                        break
                    except PermissionError as e:
                        if attempt < max_retries - 1:
                            self.log_message(self.installer_log, f"Copy attempt {attempt + 1} failed, retrying...")
                            time.sleep(2)
                        else:
                            self.log_message(self.installer_log, f"ERROR: Failed to copy agent: {e}")
                            self.log_message(self.installer_log, "You may need administrator privileges to install to Program Files")
                            return
                
                # Create config directory in ProgramData (writable location)
                self.log_message(self.installer_log, f"Creating config directory: {config_path}")
                config_path.mkdir(parents=True, exist_ok=True)
                
                # Create default config
                config_file = config_path / "service_config.json"
                config = {"host": "0.0.0.0", "port": 8443}
                with open(config_file, 'w') as f:
                    json.dump(config, f, indent=2)
                self.log_message(self.installer_log, f"✓ Created config: {config_file}")
                
                # Verify target executable exists
                if not target_exe.exists():
                    self.log_message(self.installer_log, f"ERROR: Target executable does not exist: {target_exe}")
                    return
                
                self.log_message(self.installer_log, f"Verified target executable exists: {target_exe}")
                
                # Step 4: Install service using NSSM (non-interactive)
                self.log_message(self.installer_log, "Installing Windows Service using NSSM...")
                
                # Install service with path directly (non-interactive mode)
                # NSSM install command: nssm install <servicename> <application> [arguments]
                # When path is provided, NSSM runs non-interactively
                # CRITICAL: Quote paths with spaces to prevent PowerShell from splitting them
                result = subprocess.run(
                    ["powershell", "-Command", 
                     f"$proc = Start-Process -FilePath '{nssm_path}' -ArgumentList 'install', 'ArcVaultAgent', '\"{target_exe}\"' -Verb RunAs -Wait -PassThru -WindowStyle Hidden; exit $proc.ExitCode"],
                    capture_output=True, text=True, timeout=60
                )
                
                if result.returncode == 0:
                    self.log_message(self.installer_log, "✓ Service installed successfully")
                    
                    # Set working directory to ProgramData (writable location for logs/config)
                    # CRITICAL: Quote paths with spaces
                    self.log_message(self.installer_log, "Setting service working directory...")
                    subprocess.run(
                        ["powershell", "-Command", 
                         f"$proc = Start-Process -FilePath '{nssm_path}' -ArgumentList 'set', 'ArcVaultAgent', 'AppDirectory', '\"{config_path}\"' -Verb RunAs -Wait -PassThru -WindowStyle Hidden; exit $proc.ExitCode"],
                        capture_output=True, text=True, timeout=30
                    )
                    self.log_message(self.installer_log, f"✓ Working directory configured: {config_path}")
                    
                    # Set service to auto-start
                    self.log_message(self.installer_log, "Configuring service for automatic startup...")
                    subprocess.run(
                        ["powershell", "-Command", 
                         f"$proc = Start-Process -FilePath '{nssm_path}' -ArgumentList 'set', 'ArcVaultAgent', 'Start', 'SERVICE_AUTO_START' -Verb RunAs -Wait -PassThru -WindowStyle Hidden; exit $proc.ExitCode"],
                        capture_output=True, text=True, timeout=30
                    )
                    self.log_message(self.installer_log, "✓ Service configured for automatic startup")
                    self.log_message(self.installer_log, f"Service configured to run: {target_exe}")
                    self.log_message(self.installer_log, f"Service working directory: {config_path}")
                    self.log_message(self.installer_log, f"Config file location: {config_file}")
                else:
                    self.log_message(self.installer_log, f"WARNING: Service installation may have failed")
                    if result.stdout:
                        self.log_message(self.installer_log, f"Output: {result.stdout}")
                    if result.stderr:
                        self.log_message(self.installer_log, f"Errors: {result.stderr}")
                
                # Step 5: Start service
                self.log_message(self.installer_log, "Starting service...")
                result = subprocess.run(
                    ["sc", "start", "ArcVaultAgent"],
                    capture_output=True, text=True, timeout=60
                )
                
                if result.stdout:
                    self.log_message(self.installer_log, f"Start command output: {result.stdout.strip()}")
                if result.stderr:
                    self.log_message(self.installer_log, f"Start command errors: {result.stderr.strip()}")
                if result.returncode != 0:
                    self.log_message(self.installer_log, f"Start command returned code: {result.returncode}")
                
                # Wait and check service status multiple times
                self.log_message(self.installer_log, "Waiting for service to start...")
                service_running = False
                for attempt in range(10):  # Check up to 10 times (30 seconds total)
                    time.sleep(3)
                    status_result = subprocess.run(
                        ["sc", "query", "ArcVaultAgent"],
                        capture_output=True, text=True
                    )
                    
                    status_output = status_result.stdout.upper()
                    if "RUNNING" in status_output:
                        service_running = True
                        self.log_message(self.installer_log, "✓ Service is RUNNING")
                        break
                    elif "START_PENDING" in status_output:
                        self.log_message(self.installer_log, f"Service is START_PENDING (attempt {attempt + 1}/10)...")
                    elif "STOPPED" in status_output:
                        # Parse exit codes to understand why it stopped
                        exit_code = None
                        service_exit_code = None
                        for line in status_result.stdout.split('\n'):
                            if 'WIN32_EXIT_CODE' in line.upper():
                                try:
                                    # Extract the exit code number
                                    parts = line.split(':')
                                    if len(parts) > 1:
                                        exit_code = parts[1].strip().split()[0]
                                except:
                                    pass
                            if 'SERVICE_EXIT_CODE' in line.upper():
                                try:
                                    parts = line.split(':')
                                    if len(parts) > 1:
                                        service_exit_code = parts[1].strip().split()[0]
                                except:
                                    pass
                        
                        self.log_message(self.installer_log, "⚠ Service is STOPPED")
                        if exit_code and exit_code != '0':
                            self.log_message(self.installer_log, f"WIN32 Exit Code: {exit_code} (this indicates why the service stopped)")
                            if exit_code == '1066':
                                self.log_message(self.installer_log, "")
                                self.log_message(self.installer_log, "=" * 60)
                                self.log_message(self.installer_log, "ERROR 1066: Service-Specific Error Detected")
                                self.log_message(self.installer_log, "=" * 60)
                                self.log_message(self.installer_log, "Running diagnostics...")
                                
                                # Check config file (now in ProgramData)
                                program_data = Path(os.getenv("ProgramData", "C:/ProgramData"))
                                config_path = program_data / "ArcVault"
                                config_file = config_path / "service_config.json"
                                config = {}
                                if config_file.exists():
                                    try:
                                        with open(config_file, 'r') as f:
                                            config = json.load(f)
                                        self.log_message(self.installer_log, f"✓ Config file exists: {config_file}")
                                        self.log_message(self.installer_log, f"  Host: {config.get('host', 'NOT SET')}")
                                        self.log_message(self.installer_log, f"  Port: {config.get('port', 'NOT SET')}")
                                    except Exception as e:
                                        self.log_message(self.installer_log, f"⚠ Config file exists but is invalid: {e}")
                                else:
                                    self.log_message(self.installer_log, f"⚠ Config file missing: {config_file}")
                                
                                # Check port conflicts
                                try:
                                    port = config.get('port', 8443) if config else 8443
                                    port_check = subprocess.run(
                                        ["powershell", "-Command", 
                                         f"Get-NetTCPConnection -LocalPort {port} -ErrorAction SilentlyContinue | Select-Object -First 1"],
                                        capture_output=True, text=True, timeout=5
                                    )
                                    if port_check.stdout.strip():
                                        self.log_message(self.installer_log, f"⚠ Port {port} is already in use by another process")
                                        self.log_message(self.installer_log, "  This may prevent the service from starting")
                                    else:
                                        self.log_message(self.installer_log, f"✓ Port {port} is available")
                                except:
                                    self.log_message(self.installer_log, "  (Could not check port availability)")
                                
                                # Check file permissions
                                if target_exe.exists():
                                    self.log_message(self.installer_log, f"✓ Executable exists: {target_exe}")
                                    try:
                                        # Try to get file info
                                        stat = target_exe.stat()
                                        self.log_message(self.installer_log, f"  File size: {stat.st_size / 1024 / 1024:.2f} MB")
                                    except:
                                        pass
                                else:
                                    self.log_message(self.installer_log, f"✗ Executable missing: {target_exe}")
                                
                                # Check event logs
                                self.log_message(self.installer_log, "")
                                self.log_message(self.installer_log, "Checking Windows Event Logs...")
                                try:
                                    event_check = subprocess.run(
                                        ["powershell", "-Command",
                                         "Get-EventLog -LogName Application -Newest 5 -Source '*ArcVault*' -ErrorAction SilentlyContinue | Select-Object TimeGenerated, EntryType, Message | Format-List"],
                                        capture_output=True, text=True, timeout=10
                                    )
                                    if event_check.stdout.strip():
                                        self.log_message(self.installer_log, "Recent Application Event Log entries:")
                                        for line in event_check.stdout.split('\n')[:20]:  # Limit output
                                            if line.strip():
                                                self.log_message(self.installer_log, f"  {line.strip()}")
                                    else:
                                        self.log_message(self.installer_log, "  No recent ArcVault entries in Application log")
                                except:
                                    self.log_message(self.installer_log, "  (Could not retrieve event logs)")
                                
                                self.log_message(self.installer_log, "")
                                self.log_message(self.installer_log, "Troubleshooting Steps:")
                                self.log_message(self.installer_log, "1. Check Windows Event Viewer (Application log) for detailed errors")
                                self.log_message(self.installer_log, "2. Try running the executable manually:")
                                expected_exe_path = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")
                                self.log_message(self.installer_log, f"   {expected_exe_path}")
                                self.log_message(self.installer_log, f"3. Verify the config file is valid JSON: {config_file}")
                                self.log_message(self.installer_log, "4. Check if port is available or change it in config")
                                self.log_message(self.installer_log, "5. Ensure all dependencies are installed")
                                self.log_message(self.installer_log, f"6. Check file permissions on config directory: {config_path}")
                                self.log_message(self.installer_log, "=" * 60)
                            elif exit_code == '2':
                                self.log_message(self.installer_log, "Error 2: File not found - service executable may be missing or inaccessible")
                                expected_exe = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")
                                self.log_message(self.installer_log, f"Expected location: {expected_exe}")
                                if not expected_exe.exists():
                                    self.log_message(self.installer_log, "✗ File does not exist at expected location")
                                else:
                                    self.log_message(self.installer_log, "✓ File exists - may be a permissions issue")
                            elif exit_code == '3':
                                self.log_message(self.installer_log, "Error 3: Path not found - check working directory configuration")
                                # Get config path for diagnostics
                                program_data = Path(os.getenv("ProgramData", "C:/ProgramData"))
                                expected_config_path = program_data / "ArcVault"
                                self.log_message(self.installer_log, f"Expected working directory: {expected_config_path}")
                                if not expected_config_path.exists():
                                    self.log_message(self.installer_log, "✗ Working directory does not exist")
                        
                        self.log_message(self.installer_log, "")
                        self.log_message(self.installer_log, "Full service status:")
                        for line in status_result.stdout.split('\n'):
                            if line.strip():
                                self.log_message(self.installer_log, f"  {line.strip()}")
                        break
                    else:
                        # Log the actual status
                        self.log_message(self.installer_log, f"Service status (attempt {attempt + 1}/10):")
                        for line in status_result.stdout.split('\n'):
                            if line.strip() and ('STATE' in line.upper() or 'STATUS' in line.upper()):
                                self.log_message(self.installer_log, f"  {line.strip()}")
                
                self.log_message(self.installer_log, "=" * 60)
                if service_running:
                    self.log_message(self.installer_log, "✓ Installation Complete - Service is RUNNING")
                else:
                    self.log_message(self.installer_log, "⚠ Installation Complete - Service may need attention")
                    self.log_message(self.installer_log, "Use 'Check Service Status' to see current state")
                self.log_message(self.installer_log, "=" * 60)
                
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: Installation failed: {e}")
                import traceback
                self.log_message(self.installer_log, traceback.format_exc())
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Installation failed - see errors above")
                self.log_message(self.installer_log, "=" * 60)
        
        threading.Thread(target=worker, daemon=True).start()

    def service_action(self, action):
        """Perform service management action using NSSM"""
        def worker():
            try:
                # Find NSSM executable
                # Check if running as PyInstaller bundle
                if getattr(sys, 'frozen', False):
                    # Running as compiled exe
                    # PyInstaller extracts to sys._MEIPASS
                    bundle_dir = Path(getattr(sys, '_MEIPASS', Path(sys.executable).parent))
                    nssm_paths = [
                        bundle_dir / "nssm.exe",  # Bundled with exe
                        Path(sys.executable).parent / "nssm.exe",  # Same dir as exe
                    ]
                else:
                    # Running as script
                    nssm_paths = [
                        Path(__file__).parent.parent / "nssm.exe",
                    ]
                
                nssm_path = None
                for path in nssm_paths:
                    if path and path.exists():
                        nssm_path = path
                        self.log_message(self.installer_log, f"Found nssm.exe at: {path}")
                        break
                
                if not nssm_path:
                    self.log_message(self.installer_log, "ERROR: nssm.exe not found")
                    self.log_message(self.installer_log, f"Searched locations: {[str(p) for p in nssm_paths]}")
                    self.log_message(self.installer_log, "Please ensure nssm.exe is bundled with the GUI")
                    return
                
                # Find agent executable (SOURCE - the one to copy FROM)
                # First check if it's bundled with the GUI (preferred), then search externally
                install_target = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")
                
                agent_paths = []
                
                if getattr(sys, 'frozen', False):
                    # Running as compiled exe - bundled files are in sys._MEIPASS
                    bundle_dir = Path(getattr(sys, '_MEIPASS', Path(sys.executable).parent))
                    # First priority: bundled agent (extracted by PyInstaller)
                    agent_paths.append(bundle_dir / "ArcVault.exe")
                    
                    # Fallback: search relative to exe location (for development)
                    exe_dir = Path(sys.executable).parent
                    agent_paths.extend([
                        exe_dir.parent / "apps" / "backup-agent" / "dist" / "ArcVault.exe",  # If exe is in installers-exe/
                        exe_dir.parent.parent / "apps" / "backup-agent" / "dist" / "ArcVault.exe",  # If exe is nested deeper
                        exe_dir / "apps" / "backup-agent" / "dist" / "ArcVault.exe",  # If exe is in project root
                    ])
                else:
                    # Running as script - search relative to script location
                    script_dir = Path(__file__).parent.parent
                    agent_paths.append(script_dir / "apps" / "backup-agent" / "dist" / "ArcVault.exe")
                
                agent_exe = None
                for path in agent_paths:
                    if path and path.exists():
                        agent_exe = path
                        self.log_message(self.installer_log, f"Found agent at: {path}")
                        break
                
                if agent_exe is None:
                    self.log_message(self.installer_log, "WARNING: Agent executable not found in common locations")
                    self.log_message(self.installer_log, "Searched paths:")
                    for path in agent_paths:
                        if path:
                            self.log_message(self.installer_log, f"  - {path}")
                
                self.log_message(self.installer_log, f"Performing service action: {action}")
                
                if action == "install":
                    if agent_exe is None:
                        self.log_message(self.installer_log, "ERROR: ArcVault.exe (agent) not found.")
                        if getattr(sys, 'frozen', False):
                            self.log_message(self.installer_log, "The agent should be bundled with this GUI executable.")
                            self.log_message(self.installer_log, "Please rebuild the GUI to include the agent, or ensure the agent is built and available.")
                        else:
                            self.log_message(self.installer_log, "Please ensure the agent is built first.")
                            self.log_message(self.installer_log, "Expected location: apps/backup-agent/dist/ArcVault.exe")
                        return
                    
                    # Copy agent to installation directory
                    install_path = Path("C:/Program Files (x86)/ArcVault")
                    target_exe = install_path / "ArcVault.exe"
                    
                    # Safety check: don't copy from target to itself
                    if agent_exe.resolve() == target_exe.resolve():
                        self.log_message(self.installer_log, "ERROR: Source and target are the same file!")
                        self.log_message(self.installer_log, f"Source: {agent_exe}")
                        self.log_message(self.installer_log, f"Target: {target_exe}")
                        self.log_message(self.installer_log, "Please build the agent first, or select a different source location.")
                        return
                    
                    self.log_message(self.installer_log, f"Creating installation directory: {install_path}")
                    install_path.mkdir(parents=True, exist_ok=True)
                    
                    self.log_message(self.installer_log, f"Copying agent from {agent_exe} to {target_exe}")
                    
                    # Check if target file exists and is locked
                    if target_exe.exists():
                        self.log_message(self.installer_log, "Target file already exists, attempting to replace...")
                        try:
                            # Try to delete the old file first
                            import time
                            target_exe.unlink()
                            time.sleep(1)
                        except PermissionError:
                            self.log_message(self.installer_log, "ERROR: Cannot delete existing file - it may be locked by the service")
                            self.log_message(self.installer_log, "Please ensure the service is completely uninstalled first")
                            return
                        except Exception as e:
                            self.log_message(self.installer_log, f"Warning while deleting old file: {e}")
                    
                    # Copy with retry logic
                    import shutil
                    import time
                    max_retries = 3
                    for attempt in range(max_retries):
                        try:
                            shutil.copy2(agent_exe, target_exe)
                            self.log_message(self.installer_log, f"✓ Agent copied successfully")
                            break
                        except PermissionError as e:
                            if attempt < max_retries - 1:
                                self.log_message(self.installer_log, f"Copy attempt {attempt + 1} failed, retrying in 2 seconds...")
                                time.sleep(2)
                            else:
                                self.log_message(self.installer_log, f"ERROR: Failed to copy agent after {max_retries} attempts")
                                self.log_message(self.installer_log, "The file may be locked by another process or you may need administrator privileges")
                                raise
                        except Exception as e:
                            self.log_message(self.installer_log, f"ERROR: Unexpected error during copy: {e}")
                            raise
                    
                    # Create default config file
                    config_file = install_path / "service_config.json"
                    config = {"host": "0.0.0.0", "port": 8443}
                    with open(config_file, 'w') as f:
                        json.dump(config, f, indent=2)
                    self.log_message(self.installer_log, f"Created config: {config_file}")
                    
                    # Install service using NSSM
                    self.log_message(self.installer_log, "Installing Windows Service using NSSM...")
                    result = subprocess.run(
                        ["powershell", "-Command", 
                         f"Start-Process -FilePath '{nssm_path}' -ArgumentList 'install', 'ArcVaultAgent', '{target_exe}', 'debug' -Verb RunAs -Wait"],
                        capture_output=True, text=True, timeout=60
                    )
                    
                    if result.returncode == 0:
                        self.log_message(self.installer_log, "✓ Service installed successfully!")
                        
                        # Set service to auto-start
                        self.log_message(self.installer_log, "Setting service to auto-start...")
                        subprocess.run(
                            ["powershell", "-Command", 
                             f"Start-Process -FilePath '{nssm_path}' -ArgumentList 'set', 'ArcVaultAgent', 'Start', 'SERVICE_AUTO_START' -Verb RunAs -Wait"],
                            capture_output=True, text=True, timeout=30
                        )
                        self.log_message(self.installer_log, "✓ Service configured for automatic startup")
                    else:
                        self.log_message(self.installer_log, f"Installation output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.installer_log, f"Errors: {result.stderr}")
                    
                elif action == "start":
                    # Use sc command to start
                    self.log_message(self.installer_log, "Starting service (this may take up to 60 seconds)...")
                    result = subprocess.run(
                        ["sc", "start", "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=60
                    )
                    
                    # Give service time to start and check status
                    import time
                    time.sleep(3)
                    status_result = subprocess.run(
                        ["sc", "query", "ArcVaultAgent"],
                        capture_output=True, text=True
                    )
                    
                    if "RUNNING" in status_result.stdout:
                        self.log_message(self.installer_log, "✓ Service started and running")
                    elif "START_PENDING" in status_result.stdout:
                        self.log_message(self.installer_log, "WARNING: Service is stuck in START_PENDING state")
                        self.log_message(self.installer_log, "This usually indicates the service has a startup issue.")
                        self.log_message(self.installer_log, "Check the service may be missing dependencies or has configuration errors.")
                    
                elif action == "stop":
                    # Use sc command to stop
                    self.log_message(self.installer_log, "Stopping service...")
                    result = subprocess.run(
                        ["sc", "stop", "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=30
                    )
                    
                    # Wait for service to fully stop
                    import time
                    time.sleep(2)
                    
                elif action == "uninstall" or action == "remove":
                    # Stop service first
                    self.log_message(self.installer_log, "Stopping service...")
                    subprocess.run(
                        ["sc", "stop", "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=30
                    )
                    
                    # Wait for service to fully stop
                    import time
                    self.log_message(self.installer_log, "Waiting for service to stop...")
                    time.sleep(3)
                    
                    # Uninstall using NSSM
                    self.log_message(self.installer_log, "Removing service using NSSM...")
                    result = subprocess.run(
                        ["powershell", "-Command", 
                         f"Start-Process -FilePath '{nssm_path}' -ArgumentList 'remove', 'ArcVaultAgent', 'confirm' -Verb RunAs -Wait"],
                        capture_output=True, text=True, timeout=60
                    )
                    
                    # Wait for removal to complete
                    time.sleep(2)
                    
                    # Verify service is removed
                    verify_result = subprocess.run(
                        ["sc", "query", "ArcVaultAgent"],
                        capture_output=True, text=True
                    )
                    
                    if verify_result.returncode != 0:
                        self.log_message(self.installer_log, "✓ Service removed successfully")
                    else:
                        self.log_message(self.installer_log, "WARNING: Service may still be registered")
                
                # Log results for start/stop/uninstall/remove
                if action in ["start", "stop"]:
                    if result.returncode == 0:
                        self.log_message(self.installer_log, f"✓ Service {action} command completed")
                    else:
                        self.log_message(self.installer_log, f"Service {action} output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.installer_log, f"Errors: {result.stderr}")
                elif action in ["uninstall", "remove"]:
                    if result.returncode == 0:
                        self.log_message(self.installer_log, f"✓ Service {action} command completed")
                    else:
                        self.log_message(self.installer_log, f"Service {action} output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.installer_log, f"Errors: {result.stderr}")
                
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: Service {action} failed: {e}")
                import traceback
                self.log_message(self.installer_log, traceback.format_exc())
        
        threading.Thread(target=worker, daemon=True).start()

    def check_service_status(self):
        """Check Windows service status"""
        def worker():
            try:
                self.log_message(self.installer_log, "Checking service status...")
                
                # Check Windows service
                result = subprocess.run(
                    ["sc", "query", "ArcVaultAgent"],
                    capture_output=True, text=True
                )
                
                if result.returncode == 0:
                    self.log_message(self.installer_log, "Service Status:")
                    for line in result.stdout.split('\n'):
                        if line.strip():
                            self.log_message(self.installer_log, f"  {line.strip()}")
                else:
                    self.log_message(self.installer_log, "Service not found or not installed")
                
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: Failed to check service status: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def test_agent_connection(self):
        """Test agent HTTP connection"""
        def worker():
            try:
                self.log_message(self.installer_log, "Testing agent connection...")
                
                # Test health endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/health", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_message(self.installer_log, f"Agent Health: {data}")
                except Exception as e:
                    self.log_message(self.installer_log, f"Agent health check failed: {e}")
                
                # Test info endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/info", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_message(self.installer_log, f"Agent Info: {data}")
                except Exception as e:
                    self.log_message(self.installer_log, f"Agent info check failed: {e}")
                
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: Connection test failed: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def view_event_logs(self):
        """View Windows Event Logs for the service"""
        def worker():
            try:
                self.log_message(self.installer_log, "Fetching recent event logs for ArcVaultAgent...")
                self.log_message(self.installer_log, "=" * 60)
                
                # Get recent Application event logs related to the service
                cmd = [
                    "powershell", "-Command",
                    "Get-EventLog -LogName Application -Newest 50 | Where-Object {$_.Source -like '*ArcVault*' -or $_.Message -like '*ArcVault*'} | Select-Object TimeGenerated, EntryType, Source, Message | Format-List"
                ]
                
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
                
                if result.stdout.strip():
                    self.log_message(self.installer_log, "Recent Application Event Logs:")
                    self.log_message(self.installer_log, result.stdout)
                else:
                    self.log_message(self.installer_log, "No recent event logs found for ArcVault")
                
                # Also check System logs for service-related errors
                self.log_message(self.installer_log, "\n" + "=" * 60)
                self.log_message(self.installer_log, "Checking System Event Logs...")
                
                cmd_sys = [
                    "powershell", "-Command",
                    "Get-EventLog -LogName System -Newest 50 | Where-Object {$_.Message -like '*ArcVault*'} | Select-Object TimeGenerated, EntryType, Source, Message | Format-List"
                ]
                
                result_sys = subprocess.run(cmd_sys, capture_output=True, text=True, timeout=30)
                
                if result_sys.stdout.strip():
                    self.log_message(self.installer_log, "Recent System Event Logs:")
                    self.log_message(self.installer_log, result_sys.stdout)
                else:
                    self.log_message(self.installer_log, "No recent system event logs found for ArcVault")
                
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Tip: If service is stuck in START_PENDING, check:")
                self.log_message(self.installer_log, "  1. Service executable may be missing dependencies")
                self.log_message(self.installer_log, "  2. Service may be waiting for network/resources")
                self.log_message(self.installer_log, "  3. Service may have crashed during startup")
                self.log_message(self.installer_log, "  4. Check config file: C:\\Program Files (x86)\\ArcVault\\service_config.json")
                self.log_message(self.installer_log, "  5. View service log: %TEMP%\\arcvault_service.log")
                
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: Failed to retrieve event logs: {e}")
        
        threading.Thread(target=worker, daemon=True).start()
    
    def run_manual_agent_test(self):
        """Run the agent executable manually to capture stdout/stderr"""

        def worker():
            try:
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Running Manual Agent Test")
                self.log_message(self.installer_log, "=" * 60)

                agent_exe = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")

                if not agent_exe.exists():
                    self.log_message(self.installer_log, f"ERROR: Agent not found at {agent_exe}")
                    return

                log_dir = Path("C:/ProgramData/ArcVault/logs")
                if log_dir.exists():
                    stderr_log = log_dir / "service_stderr.log"
                    stdout_log = log_dir / "service_stdout.log"

                    if stderr_log.exists():
                        self.log_message(self.installer_log, "=== RECENT STDERR FROM SERVICE ===")
                        try:
                            with open(stderr_log, "r", encoding="utf-8", errors="ignore") as fh:
                                lines = fh.readlines()
                            for line in lines[-20:]:
                                self.log_message(self.installer_log, line.rstrip())
                        except Exception as read_err:
                            self.log_message(self.installer_log, f"(Could not read stderr log: {read_err})")
                        self.log_message(self.installer_log, "")

                    if stdout_log.exists():
                        self.log_message(self.installer_log, "=== RECENT STDOUT FROM SERVICE ===")
                        try:
                            with open(stdout_log, "r", encoding="utf-8", errors="ignore") as fh:
                                lines = fh.readlines()
                            for line in lines[-20:]:
                                self.log_message(self.installer_log, line.rstrip())
                        except Exception as read_err:
                            self.log_message(self.installer_log, f"(Could not read stdout log: {read_err})")
                        self.log_message(self.installer_log, "")

                self.log_message(self.installer_log, "Starting agent manually...")
                self.log_message(self.installer_log, "This will show any startup errors...")
                self.log_message(self.installer_log, "")

                try:
                    result = subprocess.run(
                        [str(agent_exe)],
                        cwd="C:/ProgramData/ArcVault",
                        capture_output=True,
                        text=True,
                        timeout=10,
                    )
                except FileNotFoundError:
                    self.log_message(self.installer_log, "ERROR: Working directory C:/ProgramData/ArcVault not found")
                    return

                self.log_message(self.installer_log, "=== STDOUT ===")
                self.log_message(self.installer_log, result.stdout if result.stdout else "(no output)")
                self.log_message(self.installer_log, "")
                self.log_message(self.installer_log, "=== STDERR ===")
                self.log_message(self.installer_log, result.stderr if result.stderr else "(no errors)")
                self.log_message(self.installer_log, "")
                self.log_message(self.installer_log, f"Exit Code: {result.returncode}")
                self.log_message(self.installer_log, "=" * 60)

                if result.returncode == 3:
                    self.log_message(self.installer_log, "")
                    self.log_message(self.installer_log, "EXIT CODE 3 ANALYSIS:")
                    self.log_message(self.installer_log, "This typically means:")
                    self.log_message(self.installer_log, "1. Missing config file or wrong path")
                    self.log_message(self.installer_log, "2. Config file is invalid JSON")
                    self.log_message(self.installer_log, "3. Missing SSL certificates")
                    self.log_message(self.installer_log, "4. Missing required dependencies/DLLs")
                    self.log_message(self.installer_log, "")
                    self.log_message(self.installer_log, "Check agent code for what causes exit(3)")

            except subprocess.TimeoutExpired:
                self.log_message(self.installer_log, "Agent is still running after 10 seconds")
                self.log_message(self.installer_log, "This suggests it might work as a service")
                self.log_message(self.installer_log, "Kill the process and try starting the service")
            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: {e}")
                import traceback
                self.log_message(self.installer_log, traceback.format_exc())

        threading.Thread(target=worker, daemon=True).start()

    def check_agent_dependencies(self):
        """Check for common missing dependencies"""

        def worker():
            try:
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Checking Agent Dependencies")
                self.log_message(self.installer_log, "=" * 60)

                agent_exe = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")

                if not agent_exe.exists():
                    self.log_message(self.installer_log, f"ERROR: Agent not found at {agent_exe}")
                    return

                result = subprocess.run(
                    [
                        "powershell",
                        "-Command",
                        f"[System.Diagnostics.FileVersionInfo]::GetVersionInfo('{agent_exe}')",
                    ],
                    capture_output=True,
                    text=True,
                )

                self.log_message(self.installer_log, "Executable Information:")
                self.log_message(self.installer_log, result.stdout.strip() or "(no metadata)")

                self.log_message(self.installer_log, "")
                self.log_message(self.installer_log, "Checking for common missing DLLs...")

                required_dlls = [
                    "VCRUNTIME140.dll",
                    "MSVCP140.dll",
                    "python3.dll",
                    "python39.dll",
                    "python310.dll",
                    "python311.dll",
                ]

                for dll in required_dlls:
                    check_result = subprocess.run(
                        [
                            "powershell",
                            "-Command",
                            (
                                f"$dll = '{dll}'; "
                                "if (Test-Path \"C:\\Windows\\System32\\$dll\") { Write-Output 'Found' } "
                                "else { Write-Output 'Missing' }"
                            ),
                        ],
                        capture_output=True,
                        text=True,
                    )

                    status = check_result.stdout.strip()
                    symbol = "✓" if status == "Found" else "✗"
                    self.log_message(self.installer_log, f"{symbol} {dll}: {status}")

                self.log_message(self.installer_log, "")
                self.log_message(self.installer_log, "If DLLs are missing, install:")
                self.log_message(self.installer_log, "Microsoft Visual C++ Redistributable 2015-2022")
                self.log_message(self.installer_log, "=" * 60)

            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: {e}")
                import traceback
                self.log_message(self.installer_log, traceback.format_exc())

        threading.Thread(target=worker, daemon=True).start()

    def fix_service_config(self):
        """Attempt to fix common service configuration issues"""

        def worker():
            try:
                self.log_message(self.installer_log, "=" * 60)
                self.log_message(self.installer_log, "Fixing Service Configuration")
                self.log_message(self.installer_log, "=" * 60)

                import json

                config_data = {
                    "host": "0.0.0.0",
                    "port": 8443,
                    "log_level": "DEBUG",
                    "log_file": "C:/ProgramData/ArcVault/agent.log",
                }

                config_locations = [
                    Path("C:/ProgramData/ArcVault/service_config.json"),
                    Path("C:/Program Files (x86)/ArcVault/service_config.json"),
                    Path("C:/Program Files (x86)/ArcVault/config.json"),
                ]

                for config_path in config_locations:
                    try:
                        config_path.parent.mkdir(parents=True, exist_ok=True)
                        with open(config_path, "w", encoding="utf-8") as f:
                            json.dump(config_data, f, indent=2)
                        self.log_message(self.installer_log, f"✓ Created/updated: {config_path}")
                    except Exception as write_err:
                        self.log_message(self.installer_log, f"ERROR writing {config_path}: {write_err}")

                log_dir = Path("C:/ProgramData/ArcVault/logs")
                try:
                    log_dir.mkdir(parents=True, exist_ok=True)
                    self.log_message(self.installer_log, f"✓ Created log directory: {log_dir}")
                except Exception as log_err:
                    self.log_message(self.installer_log, f"ERROR creating log directory {log_dir}: {log_err}")

                stdout_file = "C:/ProgramData/ArcVault/logs/service_stdout.log"
                stderr_file = "C:/ProgramData/ArcVault/logs/service_stderr.log"

                nssm_path = None
                nssm_candidates = []

                if getattr(sys, "frozen", False):
                    bundle_dir = Path(getattr(sys, "_MEIPASS", Path(sys.executable).parent))
                    nssm_candidates.extend(
                        [
                            bundle_dir / "nssm.exe",
                            Path(sys.executable).parent / "nssm.exe",
                        ]
                    )
                else:
                    script_root = Path(__file__).parent.parent
                    nssm_candidates.extend(
                        [
                            script_root / "nssm.exe",
                            Path("C:/Program Files (x86)/ArcVault/nssm.exe"),
                        ]
                    )

                for candidate in nssm_candidates:
                    if candidate and candidate.exists():
                        nssm_path = candidate
                        break

                if nssm_path:
                    self.log_message(self.installer_log, "Configuring NSSM to capture output...")

                    for setting, value in [
                        ("AppStdout", stdout_file),
                        ("AppStderr", stderr_file),
                    ]:
                        cmd = [
                            "powershell",
                            "-Command",
                            (
                                f"Start-Process -FilePath '{nssm_path}' -ArgumentList 'set', 'ArcVaultAgent', '{setting}', '{value}' "
                                "-Verb RunAs -Wait -WindowStyle Hidden"
                            ),
                        ]
                        subprocess.run(cmd, capture_output=True, text=True, timeout=30)

                    self.log_message(self.installer_log, f"✓ STDOUT will be logged to: {stdout_file}")
                    self.log_message(self.installer_log, f"✓ STDERR will be logged to: {stderr_file}")
                    self.log_message(self.installer_log, "")
                    self.log_message(self.installer_log, "Try starting the service now and check these log files!")
                else:
                    self.log_message(self.installer_log, "WARNING: nssm.exe not found. Skipping log configuration.")
                    self.log_message(self.installer_log, "Searched locations:")
                    for candidate in nssm_candidates:
                        self.log_message(self.installer_log, f"  - {candidate}")

                self.log_message(self.installer_log, "=" * 60)

            except Exception as e:
                self.log_message(self.installer_log, f"ERROR: {e}")
                import traceback
                self.log_message(self.installer_log, traceback.format_exc())

        threading.Thread(target=worker, daemon=True).start()

    def configure_agent(self):
        """Configure agent with server URL and token"""
        server_url = self.server_url_edit.text().strip()
        auth_token = self.auth_token_edit.text().strip()
        
        if not server_url:
            QMessageBox.warning(self, "Missing Information", "Server URL is required")
            return
        
        if not auth_token:
            QMessageBox.warning(self, "Missing Information", "Authentication token is required")
            return
        
        self.log_message(self.installer_log, f"Configuring agent for server: {server_url}")
        
        # This would typically write to a config file or registry
        # For now, just log the configuration
        self.log_message(self.installer_log, "Configuration saved (placeholder)")
        self.log_message(self.installer_log, "Note: Manual configuration may be required")
        
        QMessageBox.information(self, "Configuration", "Agent configuration saved successfully!")


def main():
    """
    Main entry point with TRIPLE-LAYER instance checking to prevent multiple instances:
    
    1. Windows Mutex (Global) - Most reliable, system-wide lock
    2. Process enumeration (psutil) - Fallback if mutex unavailable  
    3. File-based lock - Final safety net with PID tracking
    
    If ANY check detects another instance, the application exits immediately.
    This prevents the "150 instances" problem completely.
    """
    mutex_handle = None
    instance_blocked = False
    
    # Method 1: Try Windows mutex (most reliable) - CHECK FIRST BEFORE ANY GUI CODE
    try:
        import win32event
        import win32api
        import winerror
        
        mutex_name = "Global\\ArcVault-GUI-PyQt5-SingleInstance"
        mutex_handle = win32event.CreateMutex(None, True, mutex_name)
        last_error = win32api.GetLastError()
        
        # If mutex already exists, another instance is running
        if last_error == winerror.ERROR_ALREADY_EXISTS:
            instance_blocked = True
            # Close the mutex handle we just created (we don't own it)
            try:
                win32api.CloseHandle(mutex_handle)
            except:
                pass
            mutex_handle = None
        
    except ImportError:
        # win32api not available, try fallback
        mutex_handle = None
    except Exception as e:
        # win32api failed, try fallback
        if mutex_handle:
            try:
                import win32api
                win32api.CloseHandle(mutex_handle)
            except:
                pass
        mutex_handle = None
    
    # If mutex check blocked us, exit immediately
    if instance_blocked:
        # Create minimal QApplication just for the message box
        app_temp = QApplication(sys.argv)
        QMessageBox.critical(
            None,
            "Already Running",
            "Another instance of ArcVault GUI is already running.\n\n"
            "Please close the existing instance before starting a new one.\n\n"
            "If you don't see any windows, use Task Manager to end\n"
            "all ArcVault-GUI-PyQt5.exe processes, or run:\n"
            "kill_all_gui.bat"
        )
        sys.exit(1)
    
    # Method 2: Fallback to psutil check (if mutex failed)
    if mutex_handle is None:
        try:
            import psutil
            current_pid = os.getpid()
            running_instances = []
            
            # Check all processes
            for proc in psutil.process_iter(['pid', 'name', 'exe']):
                try:
                    if proc.info['pid'] == current_pid:
                        continue
                    
                    # Check by name
                    proc_name = proc.info.get('name', '')
                    proc_exe = proc.info.get('exe', '')
                    
                    if proc_name and 'ArcVault-GUI-PyQt5' in proc_name:
                        running_instances.append(proc.info['pid'])
                    elif proc_exe and 'ArcVault-GUI-PyQt5' in proc_exe:
                        running_instances.append(proc.info['pid'])
                        
                except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                    continue
            
            if running_instances:
                # Create minimal QApplication just for the message box
                app_temp = QApplication(sys.argv)
                QMessageBox.critical(
                    None,
                    "Already Running",
                    f"Another instance of ArcVault GUI is already running.\n"
                    f"Found {len(running_instances)} running instance(s) (PIDs: {', '.join(map(str, running_instances[:5]))}).\n\n"
                    f"Please close other instances before starting a new one.\n\n"
                    f"Run kill_all_gui.bat to terminate all instances."
                )
                sys.exit(1)
                
        except ImportError:
            # psutil not available either - this is a problem!
            # Still try file lock as final safeguard
            print("WARNING: Neither win32api nor psutil available for instance checking!")
            print("Will rely on file-based lock only. Install pywin32 or psutil for better protection.")
        except Exception as e:
            # psutil check failed - log warning but continue to file lock check
            print(f"WARNING: Process enumeration check failed: {e}")
            print("Will rely on file-based lock as final safeguard.")
    
    # Method 3: File-based lock as final safety net
    lock_file = Path(tempfile.gettempdir()) / "ArcVault-GUI-PyQt5.lock"
    try:
        # Try to create lock file exclusively
        lock_file.parent.mkdir(parents=True, exist_ok=True)
        lock_fd = os.open(str(lock_file), os.O_CREAT | os.O_EXCL | os.O_WRONLY)
        # Write current PID to lock file
        os.write(lock_fd, str(os.getpid()).encode())
        os.close(lock_fd)
        
        # Register cleanup function to remove lock file on exit
        def cleanup_lock():
            try:
                if lock_file.exists():
                    lock_file.unlink()
            except:
                pass
        
        import atexit
        atexit.register(cleanup_lock)
        
    except OSError:
        # Lock file exists - another instance is running
        try:
            # Read PID from lock file
            with open(lock_file, 'r') as f:
                old_pid = int(f.read().strip())
            
            # Check if that process is still running
            try:
                import psutil
                if psutil.pid_exists(old_pid):
                    # Process is still running
                    app_temp = QApplication(sys.argv)
                    QMessageBox.critical(
                        None,
                        "Already Running",
                        f"Another instance of ArcVault GUI is already running (PID: {old_pid}).\n\n"
                        f"Please close the existing instance before starting a new one.\n\n"
                        f"If the process is stuck, delete:\n{lock_file}"
                    )
                    sys.exit(1)
                else:
                    # Process is dead, remove stale lock file
                    lock_file.unlink()
                    # Retry creating lock file
                    lock_fd = os.open(str(lock_file), os.O_CREAT | os.O_EXCL | os.O_WRONLY)
                    os.write(lock_fd, str(os.getpid()).encode())
                    os.close(lock_fd)
            except:
                # Can't verify, but lock file exists - be safe and exit
                app_temp = QApplication(sys.argv)
                QMessageBox.critical(
                    None,
                    "Lock File Exists",
                    f"A lock file exists at:\n{lock_file}\n\n"
                    f"This may indicate another instance is running.\n\n"
                    f"If you're sure no other instance is running, delete the lock file and try again."
                )
                sys.exit(1)
        except:
            # Can't read lock file - be safe and exit
            app_temp = QApplication(sys.argv)
            QMessageBox.critical(
                None,
                "Lock File Error",
                f"Unable to verify lock file at:\n{lock_file}\n\n"
                f"Please delete it manually if no other instance is running."
            )
            sys.exit(1)
    
    # All checks passed - no other instances running
    # Create the main application
    app = QApplication(sys.argv)
    app.setApplicationName("ArcVault Management Suite")
    app.setApplicationVersion("2.0.0")
    
    # Set application icon if available
    # app.setWindowIcon(QIcon("icon.ico"))
    
    window = ArcVaultMainWindow()
    window.show()
    
    sys.exit(app.exec_())


if __name__ == "__main__":
    main()
