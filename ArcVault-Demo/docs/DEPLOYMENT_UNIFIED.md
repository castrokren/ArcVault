# ArcVault Unified Deployment

**Everything in one package!** The unified installer combines all ArcVault components into a single, comprehensive deployment package.

## 🎯 **Why Unified?**

You're absolutely right - we need all these components anyway:
- ✅ **Client Agent** - Windows backup agents
- ✅ **Server Controller** - Cross-platform backup server  
- ✅ **Certificate Generator** - SSL certificates for HTTPS
- ✅ **JWT Generator** - Authentication keys
- ✅ **Docker Configuration** - Server deployment
- ✅ **Complete Setup** - Everything configured automatically

**Why have separate installers when one can do it all?**

## 📦 **What's Included**

### **Single Package Contains:**
```
installer-unified/
├── client/
│   ├── ArcVaultAgent.exe          # Backup agent
│   └── ArcVaultAgentService.exe   # Windows service
├── server/
│   └── ArcVaultController.exe     # Web control panel
├── tools/
│   ├── ArcVaultCertGen.exe        # Certificate generator
│   └── ArcVaultJWTGen.exe         # JWT key generator
├── docker/
│   ├── Dockerfile.server          # Server container
│   ├── docker-compose.yml         # Multi-service orchestration
│   └── nginx/nginx.conf           # HTTPS reverse proxy
├── config/
│   ├── development.env            # Development configuration
│   ├── staging.env               # Staging configuration
│   └── production.env            # Production configuration
├── ArcVault-Unified.bat          # Main launcher
├── install.bat                   # Installation script
├── setup.bat                     # Configuration wizard
├── manager.bat                   # Service management
└── README.txt                    # Documentation
```

## 🚀 **How to Deploy**

### **Step 1: Build Unified Package**
```bash
scripts\one_click_deploy.bat
# Choose option 4: Unified Installer
```

### **Step 2: Deploy to Target Machine**
1. **Copy** the entire `installer-unified` folder to target machine
2. **Run** `ArcVault-Unified.bat` as Administrator
3. **Choose** your deployment type:
   - **Client Only** - Windows backup agent
   - **Server Only** - Cross-platform backup server
   - **Both** - Complete setup

### **Step 3: Follow Setup Wizard**
The setup wizard automatically:
- ✅ **Generates certificates** for HTTPS communication
- ✅ **Creates JWT keys** for authentication
- ✅ **Configures environment** based on your choice
- ✅ **Installs Windows services** (if client)
- ✅ **Starts Docker server** (if server)
- ✅ **Sets up complete configuration**

## 🎛️ **Deployment Options**

### **Option 1: Client Only**
- Installs Windows backup agent
- Generates SSL certificates
- Configures connection to backup server
- Installs as Windows service
- **Result**: Agent ready to receive backup jobs

### **Option 2: Server Only**
- Sets up Docker server environment
- Generates SSL certificates and JWT keys
- Configures web control panel
- Starts Docker services
- **Result**: Backup server ready to manage agents

### **Option 3: Both (Complete Setup)**
- Installs Windows backup agent
- Sets up Docker server environment
- Generates all certificates and keys
- Configures complete environment
- Starts all services
- **Result**: Complete backup system ready to use

## 🔧 **Management Tools**

### **Unified Manager (`manager.bat`)**
- **Manage Windows Agent**: Start/stop/restart agent service
- **Manage Docker Server**: Start/stop/update server
- **View Service Status**: Check all service health
- **View Logs**: Access all system logs
- **Regenerate Certificates**: Update SSL certificates
- **Regenerate JWT Keys**: Update authentication keys
- **Open Web Interface**: Launch control panel

### **Service Management**
```bash
# Windows Agent
net start ArcVaultAgent
net stop ArcVaultAgent

# Docker Server
docker-compose up -d
docker-compose down
docker-compose restart
```

## 🔒 **Security Features**

### **Automatic Security Setup**
- ✅ **Self-signed certificates** generated during install
- ✅ **JWT authentication** with RSA key pairs
- ✅ **HTTPS communication** on port 443
- ✅ **Secure key storage** in ProgramData
- ✅ **Service isolation** with proper permissions

### **Certificate Management**
- **Auto-generation**: Certificates created during setup
- **Hostname-specific**: Certificates match your hostname
- **Easy renewal**: Use manager to regenerate
- **Internal use**: Self-signed for internal networks

### **JWT Authentication**
- **Production ready**: RSA256 signatures
- **Automatic rotation**: Keys generated during setup
- **Secure storage**: Keys stored in ProgramData
- **Easy management**: Regenerate through manager

## 🌐 **Access Points**

### **After Installation**
- **Web Interface**: `http://localhost:8501`
- **Agent API**: `https://localhost:443`
- **Health Check**: `https://localhost:443/health`
- **With Nginx**: `https://localhost:443` (optional)

### **Network Access**
- **Server**: `http://server-hostname:8501`
- **Clients**: `https://client-hostname:443`
- **HTTPS**: `https://server-hostname:443` (with Nginx)

## 📋 **System Requirements**

### **Windows Clients**
- **OS**: Windows 10/11 or Windows Server 2016+
- **Privileges**: Administrator for installation
- **Ports**: 443 (HTTPS)
- **No Python required**

### **Server (Any OS)**
- **Docker**: 20.10+ and Docker Compose 2.0+
- **OS**: Linux, macOS, Windows
- **Ports**: 8501 (HTTP), 443 (HTTPS with Nginx)
- **Memory**: 512MB minimum

## 🔄 **Update Process**

### **Unified Updates**
1. **Stop services** using manager
2. **Replace** installer-unified folder
3. **Run** setup.bat to reconfigure
4. **Restart** services

### **Individual Updates**
- **Agent**: Replace ArcVaultAgent.exe and restart service
- **Server**: Update Docker image and restart container
- **Certificates**: Regenerate through manager
- **JWT Keys**: Regenerate through manager

## 🚨 **Troubleshooting**

### **Installation Issues**
- **"Access denied"**: Run as Administrator
- **"Docker not found"**: Install Docker Desktop
- **"Port conflicts"**: Change ports in configuration
- **"Service won't start"**: Check logs through manager

### **Configuration Issues**
- **"Certificate errors"**: Regenerate certificates
- **"JWT errors"**: Regenerate JWT keys
- **"Connection refused"**: Check firewall and ports
- **"Service not responding"**: Check service status

### **Management Issues**
- **"Manager won't start"**: Check file permissions
- **"Docker commands fail"**: Verify Docker is running
- **"Web interface not accessible"**: Check port configuration
- **"Agent not connecting"**: Verify server URL and certificates

## 🎯 **Best Practices**

### **Deployment**
- **Test first**: Deploy to test environment
- **Backup config**: Save configuration files
- **Document changes**: Keep track of modifications
- **Monitor logs**: Check service health regularly

### **Security**
- **Use JWT**: Enable JWT authentication for production
- **Rotate keys**: Regenerate JWT keys periodically
- **Monitor access**: Check service logs for security events
- **Update regularly**: Keep certificates and keys current

### **Maintenance**
- **Regular health checks**: Use manager to monitor services
- **Log rotation**: Monitor disk space for logs
- **Service updates**: Update Docker images regularly
- **Configuration backup**: Save environment files

## 🏆 **Advantages of Unified Approach**

### **Simplicity**
- ✅ **One package** for everything
- ✅ **One installer** for all components
- ✅ **One manager** for all services
- ✅ **One configuration** for all settings

### **Efficiency**
- ✅ **No separate downloads** needed
- ✅ **No manual configuration** required
- ✅ **No missing dependencies** possible
- ✅ **No version mismatches** between components

### **Reliability**
- ✅ **Tested together** as a complete system
- ✅ **Consistent configuration** across all components
- ✅ **Unified management** reduces errors
- ✅ **Complete documentation** in one place

---

**Ready to deploy?** 
1. **Build**: Run `scripts\one_click_deploy.bat` and choose option 4
2. **Deploy**: Copy `installer-unified` folder to target machine
3. **Install**: Run `ArcVault-Unified.bat` as Administrator
4. **Configure**: Follow the setup wizard
5. **Manage**: Use `manager.bat` for ongoing management

**Everything you need, in one package!** 🎉
