# STATE — E2E Test

## Goal
Test that STATE injection works at session start.

## Invariants / decisions
None yet.

## Done
- ✅ Built new arcvault-coordinator.exe binary with WebSocket origin validation fix
- ✅ Created full ArcVault-Setup-0.5.1-windows-amd64.exe installer (31.5 MB)
  - Bundles coordinator.exe + agent.exe + Python setup wizard
  - Includes all configuration options and service installation
- ✅ Updated architecture documentation (docs/architecure/)
  - Updated version from v0.4.0 to v0.5.1
  - Added WebSocket origin validation notation
  - Updated build/deployment timestamp to June 25, 2026
- Testing STATE inject hook

## In-progress
- 

## Next
- Verify state-guard warning on session end

## Open questions
- 

## File map
- tasks/test/STATE.md — This file

