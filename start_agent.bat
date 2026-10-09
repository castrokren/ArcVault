@echo off
echo Starting ArcVault Agent...
cd /d "C:\Users\castrk05\Py_stuff\ArcVault\apps\backup-agent"

REM Set environment variables
set AGENT_PORT=8443
set AGENT_TOKEN=dev-token-12345
set AGENT_LISTEN_HOST=0.0.0.0

REM Start the agent
echo Agent starting on http://localhost:8443
C:\Users\castrk05\AppData\Local\Programs\Python\Python311\python.exe -m uvicorn agent.app:app --host 0.0.0.0 --port 8443

pause
