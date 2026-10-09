# ArcVault Backup Controller

Streamlit-based web control panel to manage backup agents and orchestrate backup jobs across your network.

## 🚀 Quick Start

### Windows (Recommended)
For comprehensive Windows deployment instructions, see **[README_WINDOWS.md](README_WINDOWS.md)**

**Quick installation:**
```powershell
# Install and configure with JWT authentication
.\scripts\install_controller.ps1

# Run the controller
python controller/run_controller.py
```

### Development Setup
```bash
python -m venv .venv
. .venv/Scripts/activate  # on Windows PowerShell: .venv\Scripts\Activate.ps1
pip install -e .
$env:AGENT_TOKEN = "dev-token"  # match your agents
streamlit run controller/app.py
```

**Access the web interface:** `http://localhost:8501`

## 📖 Documentation

- **[Windows Deployment Guide](README_WINDOWS.md)** - Comprehensive Windows installation and configuration
- **[Installation Scripts](scripts/)** - Automated setup tools
- **[Docker Deployment](Dockerfile)** - Container-based deployment

## 🔧 Key Features

- **Multi-Agent Management** - Control multiple backup agents from one interface
- **Live Log Streaming** - Real-time backup progress monitoring
- **JWT Authentication** - Secure RS256 token-based authentication
- **Web-Based Interface** - Easy-to-use Streamlit dashboard
- **Job History** - Track backup job status and history

## 🌐 Usage

1. **Add Agents** - Configure your backup agents in the web interface
2. **Create Jobs** - Select sources, destination, and backup options
3. **Monitor Progress** - Watch live logs and job status
4. **Manage History** - Review completed and failed jobs

## ⚙️ Configuration

### Basic Configuration
```powershell
$env:CONTROLLER_PORT = "8501"
$env:CONTROLLER_ADDRESS = "0.0.0.0"
$env:AGENT_TOKEN = "your-shared-token"
```

### JWT Authentication (Recommended)
```powershell
$env:CTRL_JWT_PRIVATE_KEY_PATH = "C:\ProgramData\BackupController\controller_private.pem"
$env:CTRL_JWT_ISSUER = "backup-controller"
```

### Agent URLs
Configure agents in `controller/agents.json`:
```json
{
  "server-01": "http://192.168.1.100:8443",
  "server-02": "http://192.168.1.101:8443"
}
```
