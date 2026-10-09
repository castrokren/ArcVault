# ArcVault EXE & Docker Deployment

This guide covers the new deployment options for ArcVault with self-contained EXE installers and Docker support.

## 🚀 New Deployment Features

### ✅ **Self-Contained EXE Installers**
- No Python required on target machines
- Single-file deployment
- Automated setup wizards
- Warranty-safe installation

### ✅ **HTTPS/443 Without External Certificates**
- Self-signed certificate generation
- Internal network communication
- No certificate authority required
- Browser warnings expected (internal use)

### ✅ **Docker Server Deployment**
- Cross-platform server deployment
- Linux, macOS, Windows support
- Containerized environment
- Easy updates and management

## 📦 Deployment Options

### Option 1: EXE Installers (Recommended)
```bash
# Build everything
scripts\one_click_deploy.bat
# Choose option 3: Both Packages
```

**Creates:**
- `ArcVault-Client-Installer.exe` - Client package installer
- `ArcVault-Server-Installer.exe` - Server package installer
- `ArcVault-Cert-Generator.exe` - Self-signed certificate generator
- `ArcVault-JWT-Generator.exe` - JWT key pair generator

### Option 2: Docker Server + EXE Clients
```bash
# Build client EXE installers
scripts\one_click_deploy.bat
# Choose option 1: Client Package

# Deploy server with Docker
scripts\docker_deploy.bat
```

## 🔧 Installation Process

### Client Installation (EXE)
1. **Copy installer** to client machine
2. **Run as Administrator**: `ArcVault-Client-Installer.exe`
3. **Follow setup wizard**:
   - Enter backup server URL
   - Choose authentication (token or JWT)
   - Configure SSL if needed
4. **Start agent service**

### Server Installation (EXE)
1. **Copy installer** to backup server
2. **Run as Administrator**: `ArcVault-Server-Installer.exe`
3. **Follow setup wizard**:
   - Choose environment (dev/staging/prod)
   - Enter server hostname
   - Generate JWT keys automatically
4. **Start controller**

### Server Installation (Docker)
1. **Copy Docker files** to server
2. **Run deployment script**: `scripts\docker_deploy.bat`
3. **Automatic setup**:
   - Creates data directories
   - Generates certificates and JWT keys
   - Builds and starts container
4. **Access web interface**: `http://localhost:8501`

## 🔒 Security Configuration

### Self-Signed Certificates
```bash
# Generate certificates
ArcVault-Cert-Generator.exe keys/ your-hostname

# Creates:
# - server.crt (certificate)
# - server.key (private key)
# - ca-bundle.crt (CA bundle)
```

### JWT Authentication
```bash
# Generate JWT keys
ArcVault-JWT-Generator.exe keys/

# Creates:
# - controller_private.pem (server signing key)
# - controller_public.pem (client verification key)
```

### HTTPS Configuration
- **Port**: 443 (standard HTTPS)
- **Certificates**: Self-signed (internal use)
- **Verification**: Disabled for internal networks
- **Browser warnings**: Expected (click "Advanced" → "Proceed")

## 🌐 Access Points

### EXE Deployment
- **Server**: `https://backup-server:443` (HTTPS)
- **Clients**: `https://client-machine:443` (HTTPS)
- **Controller UI**: `http://backup-server:8501` (HTTP)

### Docker Deployment
- **Server**: `http://localhost:8501` (HTTP)
- **With Nginx**: `https://localhost:443` (HTTPS)
- **Clients**: `https://client-machine:443` (HTTPS)

## 🐳 Docker Configuration

### Basic Docker Deployment
```bash
# Deploy server
scripts\docker_deploy.bat

# Access web interface
http://localhost:8501
```

### Docker with HTTPS (Nginx)
```bash
# Deploy with Nginx reverse proxy
docker-compose --profile with-nginx up -d

# Access with HTTPS
https://localhost:443
```

### Docker Management
```bash
# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Restart services
docker-compose restart

# Update services
docker-compose up -d --build
```

## 📋 System Requirements

### EXE Installers
- **Windows**: 10/11 or Server 2016+
- **Administrator privileges** for installation
- **No Python required**
- **Ports**: 443 (HTTPS), 8501 (HTTP)

### Docker Deployment
- **Docker**: 20.10+ and Docker Compose 2.0+
- **OS**: Linux, macOS, Windows
- **Ports**: 8501 (HTTP), 443 (HTTPS with Nginx)
- **Memory**: 512MB minimum

## 🔄 Update Process

### EXE Updates
1. **Stop services**
2. **Replace EXE files**
3. **Restart services**

### Docker Updates
```bash
# Update and restart
docker-compose up -d --build

# Or pull latest image
docker-compose pull
docker-compose up -d
```

## 🚨 Troubleshooting

### EXE Installation Issues
- **"Access denied"**: Run as Administrator
- **"Python not found"**: EXE installers don't need Python
- **"Port in use"**: Change ports in configuration

### Docker Issues
- **"Docker not found"**: Install Docker Desktop
- **"Port conflicts"**: Change ports in docker-compose.yml
- **"Permission denied"**: Check Docker daemon is running

### Certificate Issues
- **"Certificate warnings"**: Expected for self-signed certificates
- **"Connection refused"**: Check firewall and port settings
- **"Invalid certificate"**: Regenerate certificates

## 📊 Monitoring

### Health Checks
- **EXE Server**: `https://server:443/health`
- **Docker Server**: `http://localhost:8501`
- **Clients**: `https://client:443/health`

### Logs
- **EXE**: Windows Event Log and `ProgramData\ArcVault\logs\`
- **Docker**: `docker-compose logs -f`

## 🎯 Best Practices

### Security
- Use JWT authentication in production
- Generate unique certificates per environment
- Rotate JWT keys periodically
- Monitor service logs

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

---

**Ready to deploy?** Run `scripts\one_click_deploy.bat` to build EXE installers or `scripts\docker_deploy.bat` for Docker deployment!
