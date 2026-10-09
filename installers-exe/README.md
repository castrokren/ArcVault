# ArcVault EXE Installers

This directory contains self-contained executable installers for ArcVault backup system. These executables require no Python installation and can be run directly on Windows systems.

## 📦 Available Installers

### 🎯 ArcVault-GUI-PyQt5.exe
**Size:** ~11 MB | **Type:** Professional PyQt5 Management GUI

**NEW & IMPROVED!** Modern, professional PyQt5 interface that combines all ArcVault tools with superior styling and user experience.

### 🎯 ArcVault-GUI.exe
**Size:** ~15 MB | **Type:** Basic Tkinter Management GUI

**LEGACY:** Basic tkinter-based interface (use PyQt5 version for better experience).

**Features:**
- **SSL Certificate Generation** - Create self-signed certificates with customizable settings
- **JWT Key Generation** - Generate RSA key pairs for authentication
- **Client Installation Management** - Install and manage ArcVault clients
- **Windows Service Control** - Start, stop, and manage agent services
- **Agent Configuration** - Configure server connections and authentication
- **Connection Testing** - Test agent connectivity and health

**PyQt5 Features (Recommended):**
- **Modern Professional Interface** - Native Windows styling with advanced controls
- **Threaded Operations** - Non-blocking UI with real-time progress tracking
- **Advanced Styling** - Custom CSS-like styling for professional appearance
- **Better Performance** - More responsive and efficient than tkinter version
- **Enhanced User Experience** - Improved layouts, better error handling, and visual feedback

**Usage:**
```bash
# Simply double-click to launch (PyQt5 version recommended)
ArcVault-GUI-PyQt5.exe

# Or use legacy tkinter version
ArcVault-GUI.exe
```

**Interface Tabs:**
- **🔐 SSL Certificates** - Generate certificates with advanced options and real-time progress
- **🔑 JWT Keys** - Create authentication key pairs with threaded generation
- **💻 Client Manager** - Install, configure, and manage ArcVault clients and services
- **ℹ️ About** - Version information and comprehensive feature overview

**Benefits:**
- No command-line knowledge required
- Professional Windows application appearance
- Real-time progress indicators and threaded operations
- Comprehensive input validation and error handling
- Integrated logging and status feedback
- Modern UI controls and styling

---

### 🖥️ ArcVault-Client-Installer.exe
**Size:** ~23 MB | **Type:** NSIS Installer

Professional Windows installer for ArcVault backup agents on client machines.

**Features:**
- Modern Windows installer interface with setup wizard
- Installs to Program Files with proper permissions
- Creates desktop shortcuts and start menu entries
- Includes both agent executable and Windows service
- Auto-launches configuration wizard after installation

**Usage:**
```bash
# Run as Administrator
ArcVault-Client-Installer.exe
```

**What it installs:**
- `ArcVaultAgent.exe` - Main backup agent
- `ArcVaultAgentService.exe` - Windows service version
- Configuration templates and management scripts
- Service management tools

---

### 🔐 ArcVault-Cert-Generator.exe
**Size:** ~12 MB | **Type:** Standalone Tool

Generates self-signed SSL certificates for secure HTTPS communication.

**Usage:**
```bash
# Basic usage (creates certificates in 'keys' folder for localhost)
ArcVault-Cert-Generator.exe

# Specify output directory and hostname
ArcVault-Cert-Generator.exe C:\certs myserver.local

# Advanced options
ArcVault-Cert-Generator.exe C:\certs myserver.local --key-size 4096 --days 730 --force
```

**Parameters:**
- `output_dir` - Directory to save certificates (default: `keys`)
- `hostname` - Hostname for the certificate (default: `localhost`)
- `--key-size` - RSA key size in bits (default: `2048`)
- `--days` - Certificate validity in days (default: `365`)
- `--force` - Overwrite existing certificates

**Generated Files:**
- `server.key` - Private key (keep secure!)
- `server.crt` - SSL certificate
- `ca-bundle.crt` - Certificate authority bundle

**Example:**
```bash
# Generate certificates for production server
ArcVault-Cert-Generator.exe C:\ssl backup.company.com --key-size 4096 --days 365
```

---

### 🔑 ArcVault-JWT-Generator.exe
**Size:** ~12 MB | **Type:** Standalone Tool

Generates RSA key pairs for JWT authentication between server and clients.

**Usage:**
```bash
# Basic usage (creates keys in 'keys' folder)
ArcVault-JWT-Generator.exe

# Specify output directory
ArcVault-JWT-Generator.exe C:\jwt-keys

# Advanced options
ArcVault-JWT-Generator.exe C:\jwt-keys --key-size 4096 --force
```

**Parameters:**
- `output_dir` - Directory to save keys (default: `keys`)
- `--key-size` - RSA key size in bits (default: `2048`)
- `--force` - Overwrite existing keys

**Generated Files:**
- `controller_private.pem` - Private key for server (keep secure!)
- `controller_public.pem` - Public key for clients (can be shared)

**Example:**
```bash
# Generate 4096-bit keys for production
ArcVault-JWT-Generator.exe C:\production\keys --key-size 4096
```

---

## 🚀 Quick Start Guide

### Option 1: Using the GUI (Recommended)

**For beginners and users who prefer graphical interfaces:**

1. **Launch the GUI:**
   ```bash
   # Double-click or run from command line (PyQt5 version recommended)
   ArcVault-GUI-PyQt5.exe
   ```

2. **Generate SSL Certificates:**
   - Go to the "SSL Certificates" tab
   - Enter your server hostname (e.g., `backup-server.company.com`)
   - Choose key size (2048, 3072, or 4096 bits)
   - Set validity period (default: 365 days)
   - Click "Generate SSL Certificate"

3. **Generate JWT Keys:**
   - Go to the "JWT Keys" tab
   - Choose key size (2048, 3072, or 4096 bits)
   - Click "Generate JWT Key Pair"

4. **Install Client:**
   - Go to the "Client Installer" tab
   - Click "Run Client Installer" to install on this machine
   - Use service management buttons to control the agent
   - Configure server connection in the Quick Setup section

### Option 2: Using Command Line Tools

**For advanced users and automation:**

### 1. Server Setup
```bash
# Step 1: Generate JWT keys for authentication
ArcVault-JWT-Generator.exe C:\ArcVault\keys

# Step 2: Generate SSL certificates for HTTPS
ArcVault-Cert-Generator.exe C:\ArcVault\ssl server.company.com

# Step 3: Deploy server components (manual process)
# Copy keys and certificates to server deployment
```

### 2. Client Setup
```bash
# Step 1: Run client installer as Administrator
ArcVault-Client-Installer.exe

# Step 2: Follow setup wizard
# - Configure server connection
# - Set authentication method
# - Install Windows service

# Step 3: Copy public key from server
# Copy controller_public.pem to client keys directory
```

## 🔧 Configuration

### SSL Certificate Setup
After generating certificates with `ArcVault-Cert-Generator.exe`:

1. **Server Configuration:**
   ```env
   CONTROLLER_CERTFILE=C:\ArcVault\ssl\server.crt
   CONTROLLER_KEYFILE=C:\ArcVault\ssl\server.key
   CONTROLLER_SSL_ENABLED=true
   ```

2. **Client Configuration:**
   ```env
   AGENT_CA_BUNDLE=C:\ArcVault\ssl\ca-bundle.crt
   AGENT_VERIFY_SSL=true
   AGENT_SERVER_URL=https://server.company.com:8443
   ```

### JWT Authentication Setup
After generating keys with `ArcVault-JWT-Generator.exe`:

1. **Server Configuration:**
   ```env
   CONTROLLER_JWT_PRIVATE_KEY=C:\ArcVault\keys\controller_private.pem
   CONTROLLER_REQUIRE_JWT=true
   ```

2. **Client Configuration:**
   ```env
   AGENT_JWT_PUBLIC_KEY=C:\ArcVault\keys\controller_public.pem
   AGENT_REQUIRE_JWT=true
   ```

## 🛡️ Security Best Practices

### Certificate Security
- **Private Keys:** Store `server.key` with restricted permissions (600)
- **Certificate Files:** Readable permissions (644) are acceptable
- **Self-Signed Certs:** Suitable for internal networks, not public internet
- **Production:** Consider using certificates from trusted CAs

### JWT Key Security
- **Private Key:** Keep `controller_private.pem` secure on server only
- **Public Key:** `controller_public.pem` can be distributed to all clients
- **Key Rotation:** Regenerate keys periodically for enhanced security
- **Backup:** Securely backup private keys

### File Permissions
The generators automatically set appropriate file permissions:
- Private keys: `600` (owner read/write only)
- Public files: `644` (owner read/write, others read)

## 🔍 Troubleshooting

### Common Issues

**Certificate Generator:**
- **Unicode Error:** Files are still created successfully despite console encoding errors
- **Permission Denied:** Run as Administrator or check output directory permissions
- **Existing Files:** Use `--force` flag to overwrite existing certificates

**JWT Generator:**
- **Unicode Error:** Files are still created successfully despite console encoding errors
- **Key Size:** Larger keys (4096-bit) take longer to generate but are more secure
- **Existing Files:** Use `--force` flag to overwrite existing keys

**Client Installer:**
- **Installation Failed:** Ensure running as Administrator
- **Service Issues:** Check Windows Event Viewer for service errors
- **Connection Problems:** Verify server URL and firewall settings

### Verification

**Check Generated Certificates:**
```bash
# View certificate details (requires OpenSSL)
openssl x509 -in server.crt -text -noout

# Verify certificate and key match
openssl x509 -noout -modulus -in server.crt | openssl md5
openssl rsa -noout -modulus -in server.key | openssl md5
```

**Check Generated JWT Keys:**
```bash
# View private key details (requires OpenSSL)
openssl rsa -in controller_private.pem -text -noout

# Extract public key from private key to verify
openssl rsa -in controller_private.pem -pubout
```

## 📋 System Requirements

- **Operating System:** Windows 10/11 or Windows Server 2016+
- **Architecture:** x64 (64-bit)
- **Privileges:** Administrator rights for installation
- **Disk Space:** ~100 MB for full installation
- **Network:** HTTPS (443) and custom ports as configured

## 🔗 Related Documentation

- **ArcVault Server Setup:** See server deployment documentation
- **Client Configuration:** See agent configuration guide
- **SSL/TLS Setup:** See security configuration documentation
- **JWT Authentication:** See authentication setup guide

## 📞 Support

For issues and support:
- **GitHub:** https://github.com/castrokren/arcvault-tool
- **Documentation:** See project wiki
- **Issues:** Report bugs via GitHub Issues

---

## 📝 Build Information

These executables were built using:
- **PyInstaller:** 6.16.0
- **Python:** 3.11.9
- **NSIS:** 3.11 (for client installer)
- **Cryptography:** 46.0.3

**Build Date:** Generated automatically by build system
**Build Environment:** Windows 10 x64

---

*ArcVault - Secure, Reliable, Enterprise Backup Solution*
