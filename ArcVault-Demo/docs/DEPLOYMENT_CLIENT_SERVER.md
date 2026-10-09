# ArcVault Client & Server Deployment

This guide explains how to deploy ArcVault with separate client and server packages for production environments.

## 🏗️ Architecture Overview

ArcVault uses a client-server architecture:

- **Server (Controller)**: Manages backup jobs and provides web interface
- **Clients (Agents)**: Execute backup jobs on individual machines

## 📦 Deployment Packages

### Client Package (`deployment-client/`)
**For installing agents on client machines**

- ✅ `ArcVaultAgent.exe` - Backup agent executable
- ✅ `ArcVaultAgentService.exe` - Windows service version
- ✅ Configuration wizard for server connection
- ✅ Service management tools
- ✅ No Python required

### Server Package (`deployment-server/`)
**For installing controller on backup server**

- ✅ `ArcVaultController.exe` - Web control panel
- ✅ `ArcVaultKeyGen.exe` - JWT key generator
- ✅ Configuration wizard for server setup
- ✅ Agent management tools
- ✅ No Python required

## 🚀 Quick Deployment

### Build Packages
```bash
# Build both client and server packages
scripts\one_click_deploy.bat
# Choose option 3: Both Packages
```

### Deploy Server
1. Copy `deployment-server/` folder to backup server
2. Run `ArcVault-Server.bat`
3. Follow setup wizard
4. Start controller

### Deploy Clients
1. Copy `deployment-client/` folder to each client machine
2. Run `ArcVault-Agent.bat`
3. Configure connection to backup server
4. Start agent service

## 🔧 Installation Process

### Server Installation

#### 1. Install Server
```bash
# On backup server
ArcVault-Server.bat
# Choose: Install Server
```

#### 2. Configure Server
```bash
# Choose: Configure Server
# Select environment (development/staging/production)
# Enter server hostname
# JWT keys generated automatically
```

#### 3. Start Controller
```bash
# Choose: Manage Server
# Start Controller (Interactive or Background)
```

### Client Installation

#### 1. Install Agent
```bash
# On client machine
ArcVault-Agent.bat
# Choose: Install Agent
```

#### 2. Configure Agent
```bash
# Choose: Configure Agent
# Enter backup server URL
# Choose authentication method (token or JWT)
# Configure SSL if needed
```

#### 3. Start Service
```bash
# Choose: Manage Agent Service
# Install and start service
```

## 🌐 Access Points

### Server (Controller)
- **Web Interface**: `http://backup-server:8501`
- **API**: `http://backup-server:8501/api`

### Clients (Agents)
- **Agent API**: `http://client-machine:8443`
- **Health Check**: `http://client-machine:8443/health`
- **Agent Info**: `http://client-machine:8443/info`

## 🔒 Security Configuration

### Production Setup

#### Server (Controller)
1. **Generate JWT Keys**
   ```bash
   # Use server manager
   Manage Server > Manage JWT Keys > Generate new keys
   ```

2. **Configure SSL** (Optional)
   - Copy certificates to `ProgramData\ArcVault\keys\`
   - Files: `server.crt`, `server.key`, `ca-bundle.crt`

3. **Distribute Public Key**
   - Copy `controller_public.pem` to all client machines
   - Place in `ProgramData\ArcVault\keys\`

#### Clients (Agents)
1. **JWT Authentication**
   - Copy server's public key to `keys\controller_public.pem`
   - Configure JWT settings in setup wizard

2. **SSL Configuration** (Optional)
   - Copy certificates to `ProgramData\ArcVault\keys\`
   - Files: `agent.crt`, `agent.key`, `ca-bundle.crt`

### Development Setup
- Use simple token authentication
- HTTP (no SSL required)
- Localhost URLs

## 📋 System Requirements

### Server Machine
- Windows 10/11 or Windows Server 2016+
- Administrator privileges
- Port 8501 available
- Network access to client machines

### Client Machines
- Windows 10/11 or Windows Server 2016+
- Administrator privileges for installation
- Port 8443 available
- Network access to backup server

## 🔄 Agent Management

### Adding Agents to Server
1. Access web interface: `http://backup-server:8501`
2. Go to "Manage Agents" section
3. Add agent URL: `http://client-machine:8443`
4. Test connection
5. Configure backup jobs

### Agent Registration
1. Install agent on client machine
2. Configure connection to backup server
3. Start agent service
4. Add agent to server through web interface

## 🛠️ Management Tools

### Server Manager
- Start/stop controller
- View server status and logs
- Manage JWT keys
- Test agent connections
- Open web interface

### Agent Manager
- Install/remove Windows service
- Start/stop agent service
- View service status and logs
- Test agent connection
- Configure agent settings

## 🚨 Troubleshooting

### Common Issues

#### Server Issues
**Controller won't start**
- Check port 8501 is available
- Verify configuration in `ProgramData\ArcVault\.env`
- Check Windows Event Log

**Can't connect to agents**
- Verify agent URLs are correct
- Check network connectivity
- Ensure agents are running

#### Client Issues
**Agent service won't start**
- Check port 8443 is available
- Verify configuration in `ProgramData\ArcVault\.env`
- Check Windows Event Log

**Can't connect to server**
- Verify server URL is correct
- Check network connectivity
- Ensure server is running

### Log Locations
- **Server Logs**: `ProgramData\ArcVault\logs\`
- **Client Logs**: `ProgramData\ArcVault\logs\`
- **Service Logs**: Windows Event Log (Application)

## 📊 Monitoring

### Health Checks
- **Server**: `http://backup-server:8501`
- **Agents**: `http://client-machine:8443/health`

### Status Monitoring
- Use management tools to check service status
- Monitor backup job progress through web interface
- Check logs for errors and warnings

## 🔄 Updates

### Server Updates
1. Stop controller
2. Replace `ArcVaultController.exe`
3. Restart controller

### Client Updates
1. Stop agent service
2. Replace `ArcVaultAgent.exe`
3. Restart service

## 📞 Support

### Documentation
- Complete setup guides in deployment packages
- Troubleshooting documentation
- Security best practices

### Getting Help
- GitHub Issues: https://github.com/castrokren/arcvault-tool
- Check logs for error details
- Verify configuration settings

---

**Ready to deploy?** Run `scripts\one_click_deploy.bat` and choose option 3 to build both packages!
