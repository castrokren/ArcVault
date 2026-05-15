# ArcVault2.0 -- Project Status
**Last updated:** May 14, 2026

## Current Phase
Phase 5 COMPLETE. v0.1.0 released on GitHub.

## What works
- `coordinator.exe start` -- single binary, dashboard embedded, no separate dist folder needed
- `agent.exe` -- registers, heartbeats, polls + executes jobs, posts results
- 45 tests passing
- v0.1.0 draft release on GitHub with 10 archives (Windows/Mac/Linux, amd64/arm64)

## Release artifacts at https://github.com/castrokren/ArcVault/releases/tag/v0.1.0
- coordinator_0.1.0_windows_amd64.tar.gz -- coordinator.exe + README
- coordinator_0.1.0_darwin_amd64.tar.gz
- coordinator_0.1.0_darwin_arm64.tar.gz
- coordinator_0.1.0_linux_amd64.tar.gz
- coordinator_0.1.0_linux_arm64.tar.gz
- agent_0.1.0_windows_amd64.tar.gz -- agent.exe + agent-config.yaml + README
- agent_0.1.0_darwin_amd64.tar.gz
- agent_0.1.0_darwin_arm64.tar.gz
- agent_0.1.0_linux_amd64.tar.gz
- agent_0.1.0_linux_arm64.tar.gz
- checksums.txt

## Production deployment
1. Download coordinator archive for your platform, extract, run `coordinator init` then `coordinator start`
2. Download agent archive for each machine, edit agent-config.yaml, run `agent`
3. Open http://localhost:443 in browser

## Possible Phase 6 ideas
- Per-agent tokens instead of single admin token
- Email/webhook notifications on job failure
- `coordinator check-update` command (GitHub releases API)
- Dashboard improvements (pagination, search, theme toggle)
- Windows service / systemd unit files for auto-start
