# Backup Control (Streamlit)

Web control panel to trigger backup jobs on remote agents and follow live logs.

## Quick start

```bash
python -m venv .venv
. .venv/Scripts/activate  # on Windows PowerShell: .venv\Scripts\Activate.ps1
pip install -e .
$env:AGENT_TOKEN = "dev-token"  # match the agent
streamlit run controller/app.py
```

Set a non-default agent URL:

```powershell
$env:AGENT_URL = "http://win01:8443"
```

## Usage
- Enter sources and destination, click "Start Job".
- Click "Follow logs" to stream Server-Sent Events from the selected agent.
- Use "Refresh status" to fetch the latest job state.
