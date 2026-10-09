"""
ArcVault Main GUI Window

Integrated management interface for:
- SSL Certificate Generation
- JWT Key Generation
- Windows Service Management
"""

import sys
import os
import subprocess
import threading
import json
import urllib.request
from pathlib import Path
from datetime import datetime, timedelta
import tempfile

from PyQt5.QtWidgets import (
    QMainWindow, QTabWidget, QWidget, QVBoxLayout, QHBoxLayout, QGridLayout,
    QLabel, QLineEdit, QPushButton, QComboBox, QCheckBox, QTextEdit,
    QProgressBar, QGroupBox, QFileDialog, QMessageBox, QSplitter,
    QFrame, QScrollArea, QSpacerItem, QSizePolicy, QStatusBar
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
    print("Installing cryptography...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "cryptography"])
    from cryptography import x509
    from cryptography.x509.oid import NameOID
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import rsa
    import ipaddress


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

Valid for {self.days} days"""
            
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
- Public key: {public_file.name} (distribute to clients)"""
            
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
        self.create_service_tab()  # Service tab first (most important)
        self.create_certificate_tab()
        self.create_jwt_tab()
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

    def create_service_tab(self):
        """Create Windows Service Management tab"""
        tab = QWidget()
        self.tab_widget.addTab(tab, "⚙️ Service")
        
        # Main layout
        layout = QVBoxLayout()
        
        # Title
        title = QLabel("ArcVault Agent Service Management")
        title.setFont(QFont("Arial", 16, QFont.Bold))
        title.setAlignment(Qt.AlignCenter)
        layout.addWidget(title)
        
        # Create splitter
        splitter = QSplitter(Qt.Vertical)
        
        # Top section - Controls
        controls_widget = QWidget()
        controls_layout = QVBoxLayout()
        
        # Service Management Group
        service_group = QGroupBox("Service Management")
        service_layout = QGridLayout()
        
        # Service control buttons
        install_service_btn = QPushButton("Install Service")
        start_service_btn = QPushButton("Start Service")
        stop_service_btn = QPushButton("Stop Service")
        uninstall_service_btn = QPushButton("Uninstall Service")
        
        install_service_btn.clicked.connect(lambda: self.service_action("install"))
        start_service_btn.clicked.connect(lambda: self.service_action("start"))
        stop_service_btn.clicked.connect(lambda: self.service_action("stop"))
        uninstall_service_btn.clicked.connect(lambda: self.service_action("remove"))
        
        service_layout.addWidget(install_service_btn, 0, 0)
        service_layout.addWidget(start_service_btn, 0, 1)
        service_layout.addWidget(stop_service_btn, 1, 0)
        service_layout.addWidget(uninstall_service_btn, 1, 1)
        
        # Status and test buttons
        status_btn = QPushButton("Check Service Status")
        test_btn = QPushButton("Test Agent Connection")
        
        status_btn.clicked.connect(self.check_service_status)
        test_btn.clicked.connect(self.test_agent_connection)
        
        service_layout.addWidget(status_btn, 2, 0)
        service_layout.addWidget(test_btn, 2, 1)
        
        service_group.setLayout(service_layout)
        controls_layout.addWidget(service_group)
        
        # Configuration Group
        config_group = QGroupBox("Service Configuration")
        config_layout = QGridLayout()
        
        config_layout.addWidget(QLabel("Listen Host:"), 0, 0)
        self.service_host_edit = QLineEdit("0.0.0.0")
        config_layout.addWidget(self.service_host_edit, 0, 1)
        
        config_layout.addWidget(QLabel("Port:"), 1, 0)
        self.service_port_edit = QLineEdit("8443")
        config_layout.addWidget(self.service_port_edit, 1, 1)
        
        save_config_btn = QPushButton("Save Configuration")
        save_config_btn.clicked.connect(self.save_service_config)
        config_layout.addWidget(save_config_btn, 2, 0, 1, 2)
        
        config_group.setLayout(config_layout)
        controls_layout.addWidget(config_group)
        
        controls_widget.setLayout(controls_layout)
        splitter.addWidget(controls_widget)
        
        # Bottom section - Output Log
        log_group = QGroupBox("Output Log")
        log_layout = QVBoxLayout()
        self.service_log = QTextEdit()
        self.service_log.setMinimumHeight(200)
        log_layout.addWidget(self.service_log)
        log_group.setLayout(log_layout)
        splitter.addWidget(log_group)
        
        layout.addWidget(splitter)
        tab.setLayout(layout)

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
        settings_layout.addWidget(self.cert_country_edit, 0, 3)
        
        # Organization
        settings_layout.addWidget(QLabel("Organization:"), 1, 2)
        self.cert_org_edit = QLineEdit("ArcVault")
        settings_layout.addWidget(self.cert_org_edit, 1, 3)
        
        # Force overwrite
        self.cert_force_check = QCheckBox("Overwrite existing certificates")
        settings_layout.addWidget(self.cert_force_check, 2, 2, 1, 2)
        
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
        version = QLabel("Version 3.0.0 - Unified Edition")
        version.setFont(QFont("Arial", 12))
        version.setAlignment(Qt.AlignCenter)
        layout.addWidget(version)
        
        # Description
        desc = QLabel("All-in-one management tool for ArcVault backup system")
        desc.setFont(QFont("Arial", 10))
        desc.setAlignment(Qt.AlignCenter)
        layout.addWidget(desc)
        
        layout.addSpacing(20)
        
        # Features
        features_group = QGroupBox("Features")
        features_layout = QVBoxLayout()
        
        features = [
            "⚙️ Windows Service Management (Install/Start/Stop/Uninstall)",
            "🔐 SSL Certificate Generation",
            "🔑 JWT Key Pair Generation",
            "🔧 Service Configuration",
            "🔍 Connection Testing and health monitoring",
            "🎨 Modern PyQt5 interface",
            "📊 Real-time progress tracking and detailed logging",
            "✨ One executable does everything - no scripts needed!"
        ]
        
        for feature in features:
            label = QLabel(feature)
            label.setFont(QFont("Arial", 10))
            features_layout.addWidget(label)
        
        features_group.setLayout(features_layout)
        layout.addWidget(features_group)
        
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
        else:
            self.log_message(self.jwt_log, message)
            self.status_bar.showMessage("JWT key generation failed")
            QMessageBox.critical(self, "Error", message)

    def save_service_config(self):
        """Save service configuration to file"""
        try:
            host = self.service_host_edit.text().strip()
            port_str = self.service_port_edit.text().strip()
            
            try:
                port = int(port_str)
            except ValueError:
                QMessageBox.warning(self, "Invalid Input", "Port must be a number")
                return
            
            # Save to installation directory if it exists
            install_path = Path("C:/Program Files (x86)/ArcVault")
            if install_path.exists():
                config_file = install_path / "service_config.json"
            else:
                # Save to current directory
                config_file = Path.cwd() / "service_config.json"
            
            config = {"host": host, "port": port}
            with open(config_file, 'w') as f:
                json.dump(config, f, indent=2)
            
            self.log_message(self.service_log, f"✓ Configuration saved to: {config_file}")
            self.status_bar.showMessage(f"Configuration saved to: {config_file}")
            
        except Exception as e:
            self.log_message(self.service_log, f"ERROR: Failed to save config: {e}")
            QMessageBox.critical(self, "Error", f"Failed to save configuration: {e}")

    def service_action(self, action):
        """Perform service management action"""
        def worker():
            try:
                # Get path to this executable
                if getattr(sys, 'frozen', False):
                    # Running as compiled exe
                    exe_path = Path(sys.executable)
                else:
                    # Running as script
                    QMessageBox.warning(None, "Not Supported", 
                                      "Service installation only works with the compiled .exe\n"
                                      "Please build the application first with PyInstaller.")
                    return
                
                self.log_message(self.service_log, f"Performing service action: {action}")
                
                if action == "install":
                    # Copy to installation directory
                    install_path = Path("C:/Program Files (x86)/ArcVault")
                    self.log_message(self.service_log, f"Creating installation directory: {install_path}")
                    install_path.mkdir(parents=True, exist_ok=True)
                    
                    target_exe = install_path / "ArcVault.exe"
                    self.log_message(self.service_log, f"Copying to: {target_exe}")
                    
                    # Try to copy with retries (file might be locked)
                    import shutil
                    import time
                    max_retries = 3
                    for attempt in range(max_retries):
                        try:
                            # Delete old file first if it exists
                            if target_exe.exists():
                                self.log_message(self.service_log, f"Removing old executable (attempt {attempt + 1})...")
                                import time
                                time.sleep(2)  # Wait for file handles to release
                                target_exe.unlink()
                            
                            shutil.copy2(exe_path, target_exe)
                            self.log_message(self.service_log, "✓ Executable copied successfully")
                            break
                        except PermissionError as e:
                            if attempt < max_retries - 1:
                                self.log_message(self.service_log, f"File locked, waiting... (attempt {attempt + 1}/{max_retries})")
                                time.sleep(3)
                            else:
                                self.log_message(self.service_log, f"ERROR: Cannot copy executable - file is locked by another process")
                                self.log_message(self.service_log, "Please wait 10 seconds and try again, or restart your computer")
                                raise
                    
                    # Save config
                    config_file = install_path / "service_config.json"
                    config = {
                        "host": self.service_host_edit.text().strip(),
                        "port": int(self.service_port_edit.text().strip())
                    }
                    with open(config_file, 'w') as f:
                        json.dump(config, f, indent=2)
                    self.log_message(self.service_log, f"Created config: {config_file}")
                    
                    # Install service
                    self.log_message(self.service_log, "Installing Windows Service...")
                    result = subprocess.run(
                        ["powershell", "-Command", 
                         f"Start-Process -FilePath '{target_exe}' -ArgumentList 'install' -Verb RunAs -Wait"],
                        capture_output=True, text=True, timeout=60
                    )
                    
                    if result.returncode == 0:
                        self.log_message(self.service_log, "✓ Service installed successfully!")
                        
                        # Set auto-start using sc config (simpler and faster)
                        self.log_message(self.service_log, "Setting service to auto-start...")
                        try:
                            auto_result = subprocess.run(
                                ["sc", "config", "ArcVaultAgent", "start=auto"],
                                capture_output=True, text=True, timeout=10
                            )
                            if auto_result.returncode == 0:
                                self.log_message(self.service_log, "✓ Service configured for automatic startup")
                            else:
                                self.log_message(self.service_log, f"Note: Auto-start config returned: {auto_result.stdout}")
                        except Exception as e:
                            self.log_message(self.service_log, f"Note: Could not set auto-start: {e}")
                        
                        self.status_bar.showMessage("Service installed successfully")
                    else:
                        self.log_message(self.service_log, f"Installation output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.service_log, f"Errors: {result.stderr}")
                
                elif action in ["start", "stop"]:
                    result = subprocess.run(
                        ["sc", action, "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=30
                    )
                    
                    if result.returncode == 0:
                        self.log_message(self.service_log, f"✓ Service {action} completed successfully")
                        self.status_bar.showMessage(f"Service {action} completed")
                    else:
                        self.log_message(self.service_log, f"Output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.service_log, f"Errors: {result.stderr}")
                
                elif action == "remove":
                    installed_exe = Path("C:/Program Files (x86)/ArcVault/ArcVault.exe")
                    
                    # Stop first
                    self.log_message(self.service_log, "Stopping service...")
                    subprocess.run(
                        ["sc", "stop", "ArcVaultAgent"],
                        capture_output=True, text=True, timeout=30
                    )
                    
                    import time
                    time.sleep(2)
                    
                    # Uninstall
                    if installed_exe.exists():
                        self.log_message(self.service_log, "Uninstalling service...")
                        result = subprocess.run(
                            ["powershell", "-Command", 
                             f"Start-Process -FilePath '{installed_exe}' -ArgumentList 'remove' -Verb RunAs -Wait"],
                            capture_output=True, text=True, timeout=60
                        )
                    else:
                        # Try sc delete
                        result = subprocess.run(
                            ["sc", "delete", "ArcVaultAgent"],
                            capture_output=True, text=True, timeout=30
                        )
                    
                    if result.returncode == 0:
                        self.log_message(self.service_log, "✓ Service uninstalled successfully")
                        self.status_bar.showMessage("Service uninstalled successfully")
                    else:
                        self.log_message(self.service_log, f"Output: {result.stdout}")
                        if result.stderr:
                            self.log_message(self.service_log, f"Errors: {result.stderr}")
                
            except Exception as e:
                self.log_message(self.service_log, f"ERROR: Service {action} failed: {e}")
                import traceback
                self.log_message(self.service_log, traceback.format_exc())
        
        threading.Thread(target=worker, daemon=True).start()

    def check_service_status(self):
        """Check Windows service status"""
        def worker():
            try:
                self.log_message(self.service_log, "Checking service status...")
                
                result = subprocess.run(
                    ["sc", "query", "ArcVaultAgent"],
                    capture_output=True, text=True
                )
                
                if result.returncode == 0:
                    self.log_message(self.service_log, "Service Status:")
                    for line in result.stdout.split('\n'):
                        if line.strip():
                            self.log_message(self.service_log, f"  {line.strip()}")
                else:
                    self.log_message(self.service_log, "Service not found or not installed")
                
            except Exception as e:
                self.log_message(self.service_log, f"ERROR: Failed to check status: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

    def test_agent_connection(self):
        """Test agent HTTP connection"""
        def worker():
            try:
                self.log_message(self.service_log, "Testing agent connection...")
                
                # Test health endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/health", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_message(self.service_log, f"✓ Agent Health: {data}")
                except Exception as e:
                    self.log_message(self.service_log, f"✗ Agent health check failed: {e}")
                
                # Test info endpoint
                try:
                    with urllib.request.urlopen("http://localhost:8443/info", timeout=5) as response:
                        data = json.loads(response.read().decode())
                        self.log_message(self.service_log, f"✓ Agent Info: {data}")
                except Exception as e:
                    self.log_message(self.service_log, f"✗ Agent info check failed: {e}")
                
            except Exception as e:
                self.log_message(self.service_log, f"ERROR: Connection test failed: {e}")
        
        threading.Thread(target=worker, daemon=True).start()

