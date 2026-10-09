# ArcVault macOS App Deployment

This guide explains how to deploy the ArcVault controller stack on macOS using the packaged installer (`ArcVault-<version>.pkg`). The installer wraps the Docker-based deployment behind a native GUI so operators do not have to work with shell scripts.

## Prerequisites

- macOS 12.0 Monterey or later (Intel or Apple Silicon)
- Docker Desktop for Mac (running, with the CLI available)
- Internet access for pulling container images on first launch
- Administrator privileges to install packages

## Installation

1. Double-click `ArcVault-<version>.pkg`.
2. Follow the installer prompts. The preflight check warns if Docker Desktop is not detected.
3. On completion, the installer copies resources into:
   - `/Applications/ArcVault Manager.app`
   - `/Library/Application Support/ArcVault/`
   - `/usr/local/bin/arcvaultctl`
4. The post-install step builds a user-specific workspace under `~/Library/Application Support/ArcVault` for the signed-in macOS user and keeps the `#backup_logs` directory in that location.
5. If Docker Desktop is not already installed, the installer automatically downloads the latest official build and launches it. You can re-run the same process later via ArcVault Manager or `arcvaultctl install-docker` if needed.

## First Launch

1. Start Docker Desktop if it is not already running.
2. Open `/Applications/ArcVault Manager.app`.
3. The GUI checks Docker availability, loads the `.env` configuration, and exposes actions:
   - `Start` / `Stop` / `Restart` / `Update`
   - `Refresh Status` (health probe via `docker compose ps`)
   - `Open Web UI` (opens `http://localhost:8501`)
   - `Open Logs Folder` (opens `~/Library/Application Support/ArcVault/#backup_logs`)
   - **Install Docker Desktop** button (appears only if Docker is missing) to download the latest official build and launch it after installation.
4. Edit the environment configuration in the lower pane to adjust hostnames, ports, JWT or TLS settings, then click `Save Configuration`.

## Command-Line Control

The installer drops `arcvaultctl` into `/usr/local/bin` for scripting and automation:

```bash
arcvaultctl start      # start containers
arcvaultctl stop       # stop containers
arcvaultctl restart    # restart containers
arcvaultctl update     # pull new images and restart
arcvaultctl status     # JSON health report
arcvaultctl install-docker # download & install Docker Desktop
arcvaultctl logs      # show docker logs
arcvaultctl logs -f    # tail docker logs
arcvaultctl open       # open Streamlit UI in browser
```

`arcvaultctl` simply delegates to the same helper scripts that the GUI uses, so GUI and CLI remain in sync.

## Launch Agents (Optional Auto-Start)

During installation a LaunchAgent template is copied to `~/Library/LaunchAgents/com.arcvault.arcvault.plist`. To auto-start the stack when the user logs in, run:

```bash
launchctl load -w ~/Library/LaunchAgents/com.arcvault.arcvault.plist
```

To disable auto-start:

```bash
launchctl unload ~/Library/LaunchAgents/com.arcvault.arcvault.plist
```

Logs for the LaunchAgent and Docker stack live in `~/Library/Application Support/ArcVault/#backup_logs`.

## Updating

1. Open the new installer package and run it (the installer is idempotent).
2. Alternatively, run `arcvaultctl update` inside Terminal to pull updated images.
3. After installation, the GUI will reflect the updated version automatically.

## Troubleshooting

- **Docker not running**: The GUI and CLI return an error stating Docker Desktop is unavailable. Start Docker Desktop and retry.
- **Ports already in use**: Adjust `CONTROLLER_PORT` in `~/Library/Application Support/ArcVault/config/.env`, then restart the stack.
- **LaunchAgent not starting**: Run `launchctl bootout gui/$UID ~/Library/LaunchAgents/com.arcvault.arcvault.plist` followed by `launchctl bootstrap gui/$UID ...` to reload; check the logs in `#backup_logs`.
- **Reset environment**: Delete `~/Library/Application Support/ArcVault`, rerun the installer, or re-launch the manager (it recreates defaults as needed).

