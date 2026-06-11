# Phase 18 Design Spec — User-Friendly Deployment Package
**Date:** 2026-05-22  
**Version target:** v1.1.0  
**Status:** Approved

---

## Overview

Phase 18 wraps ArcVault in a polished, native-GUI installer for all three platforms — Windows, macOS, and Linux. The user downloads a single installer, picks which components to install (Coordinator, Agent, or both), configures them through a guided wizard, and ends up with the service running and the dashboard open in their browser. No CLI required for non-technical users; sysadmins get the same clean experience.

The VNC installer is the reference UX: one package, component selection, done.

---

## Target Audience

Both non-technical home/self-hosters and sysadmins/small teams. The installer must work without assuming any terminal knowledge, while not getting in the way of users who know what they're doing.

---

## Installer Flow (All Platforms)

1. **Welcome** — ArcVault logo, version, brief description
2. **Component Select** — Coordinator (server), Agent (client), or Both
3. **Install Path** — Platform-appropriate default, user-overridable
4. **Configuration Wizard** — Adapts to selected components:
   - *Coordinator:* Admin username, admin password (strength indicator), port (default 8080), optional HTTPS toggle
   - *Agent:* Coordinator URL, agent ID (auto-suggested from hostname), auth token (paste field)
   - *Both:* Coordinator config first, agent URL pre-filled as `localhost:8080`, token auto-generated from local coordinator
5. **Review & Install** — Summary of all choices before committing
6. **Post-Install** — Service registered and started, browser opens to `http://localhost:<port>`

---

## Platform Implementation

### Windows — NSIS (.exe)
- Toolchain: NSIS with Modern UI 2 (MUI2) — same toolkit as VNC, Git for Windows, WinSCP
- Wizard pages built with nsDialogs for custom config fields
- Post-install: exec `coordinator install-service` / `agent install-service`, open browser via `ShellExecute`
- Uninstaller registered in Add/Remove Programs
- Code signing slot reserved in goreleaser for future Authenticode signing
- Output: `ArcVault-Setup-{version}-windows-amd64.exe`

### macOS — .pkg
- Toolchain: `pkgbuild` + `productbuild` (Xcode command-line tools)
- Native Installer.app wizard: Welcome, ReadMe, License, Component Select, Install
- Post-install script: `launchctl` service registration
- SwiftUI helper app (`ArcVaultSetup.app`) bundled in pkg — launches after install, runs config wizard, writes `config.json`, opens browser
- Notarization slot reserved in goreleaser for future Apple notarization
- Output: `ArcVault-Setup-{version}-macos-arm64.pkg`, `ArcVault-Setup-{version}-macos-amd64.pkg`

### Linux — .deb + .rpm
- Toolchain: goreleaser nfpm (already partially configured)
- Post-install script runs `arcvault-setup` — a small Go binary bundled in the package
- `arcvault-setup` presents an interactive terminal wizard (appropriate for Linux users), writes `config.json`, registers systemd service, prints browser URL
- Supports Debian/Ubuntu (.deb) and RHEL/Fedora/Rocky (.rpm)
- Output: `arcvault_{version}_amd64.deb`, `arcvault_{version}_x86_64.rpm`

---

## Shared Config Wizard Logic

`cmd/setup/main.go` + `cmd/setup/wizard.go` — Go binary containing:
- Question/answer flow engine
- `config.json` writer (coordinator and agent configs)
- Token auto-generation when both components installed on same machine
- `cmd/setup/browser.go` — cross-platform browser open via `os/exec`

This binary is called by the Windows NSIS installer, the macOS post-install script, and ships as `arcvault-setup` in the Linux packages.

---

## Repo Structure

```
installer/
  windows/
    arcvault.nsi          # NSIS wizard script
    installer.ico         # App icon
    welcome.bmp           # MUI2 sidebar graphic
  macos/
    welcome.html          # Installer.app welcome page
    license.html          # License page
    distribution.xml      # productbuild component layout
    postinstall           # Shell: launchctl + open browser
    ArcVaultSetup.swift   # SwiftUI config wizard helper
  linux/
    postinst              # Debian post-install script
    postrm                # Debian uninstall cleanup
    arcvault.spec         # RPM spec hooks

cmd/
  setup/
    main.go               # Entry point
    wizard.go             # Config wizard flow + config.json writer
    browser.go            # Open browser after install
```

---

## goreleaser Changes

- `nfpm` block expanded with `.deb` + `.rpm` scripts
- `after_hooks` added: runs `makensis` (Windows), `productbuild` (macOS) after binary build
- New `archives` entry for installer artifact naming
- GitHub release assets updated to include all installer artifacts alongside existing zip/tar.gz

---

## Release Artifacts

```
ArcVault-Setup-1.1.0-windows-amd64.exe
ArcVault-Setup-1.1.0-macos-arm64.pkg
ArcVault-Setup-1.1.0-macos-amd64.pkg
arcvault_1.1.0_amd64.deb
arcvault_1.1.0_x86_64.rpm
```

---

## Version

Ships as **v1.1.0** — significant UX milestone, fully backward compatible with all v1.0.0 coordinators and agents.

---

## Success Criteria

- A non-technical user on Windows can download the `.exe`, double-click, click through the wizard, and have the dashboard open in their browser — without touching a terminal
- A sysadmin on Linux can `apt install ./arcvault_1.1.0_amd64.deb` and be guided through config in under 2 minutes
- A macOS user can double-click the `.pkg` and complete setup via the native Installer.app
- All three platforms register ArcVault as a system service that survives reboots
- Uninstall cleanly removes all files and services on all platforms

---

## Out of Scope

- GUI uninstaller wizard (uninstall via Add/Remove Programs on Windows, `apt remove` on Linux, manual on macOS)
- Auto-update of the installer itself
- Silent/enterprise deployment mode (future phase)
- Windows ARM builds
