# ArcVault Deployment: Windows Clients + Cross-Platform Server

This guide covers ArcVault deployment with Windows-only clients and cross-platform server support.

## 🏗️ Architecture

- **Clients**: Windows-only agents (EXE installers)
- **Server**: Cross-platform controller (Docker deployment)
- **Communication**: HTTPS/443 with self-signed certificates

## 📦 Deployment Packages

### Windows Client Package
- ✅ `ArcVault-Client-Installer.exe` - Self-contained Windows installer
- ✅ `ArcVault-Cert-Generator.exe` - Self-signed certificate generator
- ✅ `ArcVault-JWT-Generator.exe` - JWT key pair generator
- ✅ No Python required on client machines

### Cross-Platform Server Package
- ✅ Docker container (Linux, macOS, Windows)
- ✅ Nginx reverse proxy (optional HTTPS)
- ✅ Automated setup scripts
- ✅ Self-signed certificate generation

## 🚀 Quick Deployment

### Build Everything (Windows Development Machine)
```bash
scripts\one_click_deploy.bat
# Choose option 3: Both Packages
```

### Deploy Server (Any OS)
```bash
# Windows
scripts\docker_deploy.bat

# macOS/Linux
scripts/docker_deploy.sh
```

Or install the packaged macOS app (`ArcVault-<version>.pkg`) for a native GUI that wraps the Docker stack. See `docs/DEPLOYMENT_MACOS_APP.md` for details.

### Deploy Clients (Windows Only)
1. Copy `ArcVault-Client-Installer.exe` to Windows machines
2. Run as Administrator
3. Follow setup wizard

## 🔧 Installation Process

### Server Deployment (Cross-Platform)

#### Windows Server
```bash
# Copy Docker files to server
# Run deployment script
scripts\docker_deploy.bat

# Access web interface
http://localhost:8501
```

#### macOS/Linux Server
```bash
# Copy Docker files to server
# Run deployment script
scripts/docker_deploy.sh

# Access web interface
http://localhost:8501
```

#### Docker Features
- **Automatic setup**: Creates directories, generates certificates
- **JWT key generation**: Secure authentication
- **Nginx support**: Optional HTTPS termination
- **Easy management**: Start/stop/update with Docker commands

### Client Deployment (Windows Only)

#### Installation
1. **Copy installer** to Windows client machine
2. **Run as Administrator**: `ArcVault-Client-Installer.exe`
3. **Follow setup wizard**:
   - Enter backup server URL
   - Choose authentication (token or JWT)
   - Configure SSL settings
4. **Start agent service**

#### Configuration
- **Server URL**: `https://backup-server:443` or `http://backup-server:8501`
- **Authentication**: JWT (production) or simple token (development)
- **SSL**: Self-signed certificates (internal use)
- **Port**: 443 (HTTPS) or 8443 (HTTP)

## 🌐 Access Points

### Server (Cross-Platform)
- **Web Interface**: `http://server:8501`
- **With Nginx**: `https://server:443`
- **API**: `http://server:8501/api`

### Clients (Windows Only)
- **Agent API**: `https://client-machine:443`
- **Health Check**: `https://client-machine:443/health`
- **Agent Info**: `https://client-machine:443/info`

## 🔒 Security Configuration

### Self-Signed Certificates
```bash
# Generate certificates (Windows)
ArcVault-Cert-Generator.exe keys/ your-hostname

# Generate certificates (Docker)
python3 scripts/generate_self_signed_cert.py data/keys your-hostname
```

### JWT Authentication
```bash
# Generate JWT keys (Windows)
ArcVault-JWT-Generator.exe keys/

# Generate JWT keys (Docker)
python3 scripts/generate_jwt_keys.py data/keys
```

### HTTPS Configuration
- **Port**: 443 (standard HTTPS)
- **Certificates**: Self-signed (internal use)
- **Verification**: Disabled for internal networks
- **Browser warnings**: Expected (click "Advanced" → "Proceed")

## 📋 System Requirements

### Windows Clients
- **OS**: Windows 10/11 or Windows Server 2016+
- **Privileges**: Administrator for installation
- **Ports**: 443 (HTTPS)
- **No Python required**

### Server (Any OS)
- **Docker**: 20.10+ and Docker Compose 2.0+
- **OS**: Linux, macOS, Windows
- **Ports**: 8501 (HTTP), 443 (HTTPS with Nginx)
- **Memory**: 512MB minimum

## 🐳 Docker Management

### Basic Commands
```bash
# Start server
docker-compose up -d

# View logs
docker-compose logs -f

# Stop server
docker-compose down

# Restart server
docker-compose restart

# Update server
docker-compose up -d --build
```

### HTTPS with Nginx
```bash
# Enable HTTPS
docker-compose --profile with-nginx up -d

# Access with HTTPS
https://localhost:443
```

### Configuration Files
- **Environment**: `data/config/.env`
- **JWT Keys**: `data/keys/`
- **Certificates**: `data/keys/`
- **Logs**: `data/logs/`

## 🔄 Update Process

### Windows Clients
1. **Stop agent service**
2. **Replace EXE files**
3. **Restart service**

### Docker Server
```bash
# Update and restart
docker-compose up -d --build

# Or pull latest image
docker-compose pull
docker-compose up -d
```

## 🚨 Troubleshooting

### Windows Client Issues
- **"Access denied"**: Run as Administrator
- **"Python not found"**: EXE installers don't need Python
- **"Port in use"**: Change ports in configuration
- **"Certificate warnings"**: Expected for self-signed certificates

### Docker Server Issues
- **"Docker not found"**: Install Docker Desktop
- **"Port conflicts"**: Change ports in docker-compose.yml
- **"Permission denied"**: Check Docker daemon is running
- **"Service won't start"**: Check logs with `docker-compose logs`

### Network Issues
- **"Connection refused"**: Check firewall and port settings
- **"Certificate errors"**: Regenerate certificates
- **"JWT errors"**: Verify key files and permissions

## 📊 Monitoring

### Health Checks
- **Server**: `http://server:8501` or `https://server:443`
- **Clients**: `https://client:443/health`

### Logs
- **Windows Clients**: Windows Event Log and `ProgramData\ArcVault\logs\`
- **Docker Server**: `docker-compose logs -f`

## 🎯 Best Practices

### Security
- Use JWT authentication in production
- Generate unique certificates per environment
- Rotate JWT keys periodically
- Monitor service logs for security events

### Performance
- Use HTTPS for production communication
- Monitor resource usage
- Regular backup of configuration
- Test updates in staging first

### Maintenance
- Keep certificates up to date
- Monitor disk space for logs
- Regular service health checks
- Document configuration changes

## 🔄 Agent Management

### Adding Windows Agents
1. **Install agent** on Windows machine using EXE installer
2. **Configure connection** to backup server
3. **Start agent service**
4. **Add agent** to server through web interface
5. **Configure backup jobs**

### Agent Registration
- **Agent URL**: `https://windows-machine:443`
- **Authentication**: JWT or simple token
- **SSL**: Self-signed certificates
- **Health Check**: `https://windows-machine:443/health`

---

**Ready to deploy?** 
- **Windows**: Run `scripts\one_click_deploy.bat`
- **Server**: Use `scripts\docker_deploy.bat` (Windows) or `scripts/docker_deploy.sh` (macOS/Linux)
- **Clients**: Deploy `ArcVault-Client-Installer.exe` to Windows machines
