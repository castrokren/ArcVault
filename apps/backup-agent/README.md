# ArcVault Backup Agent

Lightweight per-machine agent that accepts backup jobs over HTTP and streams live logs.

## 🚀 Quick Start

### Windows (Recommended)
For comprehensive Windows deployment instructions, see **[README_WINDOWS.md](README_WINDOWS.md)**

**One-line installation:**
```powershell
# Run PowerShell as Administrator
.\Install-ArcVaultAgent.ps1 -AgentToken "your-token" -PublicUrl "http://your-ip:8443"
```

### Development Setup
```bash
python -m venv .venv
. .venv/Scripts/activate  # on Windows PowerShell: .venv\Scripts\Activate.ps1
pip install -e .
$env:AGENT_TOKEN = "dev-token"
uvicorn agent.app:app --host 0.0.0.0 --port 8443 --reload
```

### API Testing
```bash
# Create job
curl -X POST http://localhost:8443/jobs -H "X-API-Key: dev-token" -H "Content-Type: application/json" -d '{"sources":["C:/data"],"destination":"D:/backup"}'

# Stream logs
curl -N http://localhost:8443/jobs/<job_id>/logs -H "X-API-Key: dev-token"
```

## 📖 Documentation

- **[Windows Deployment Guide](README_WINDOWS.md)** - Comprehensive Windows installation and management
- **[Installation Scripts](../../README_INSTALLATION.md)** - Automated installation tools
- **[Management Guide](../../INSTALLATION_GUIDE.md)** - Detailed configuration and troubleshooting

## 🔧 Key Features

- **Headless Operation** - Runs as Windows Scheduled Task
- **Auto-Start** - Starts automatically with Windows
- **Live Log Streaming** - Server-Sent Events for real-time logs
- **JWT Authentication** - Secure RS256 token support
- **Easy Management** - PowerShell management scripts included

## 🔒 Production Notes

- Use JWT authentication with RS256 keys (see Windows guide)
- Set `AGENT_REQUIRE_JWT=true` for JWT-only mode
- Configure SSL/TLS for HTTPS endpoints
- Use strong, unique tokens for each agent
- Restrict network access to agent ports
