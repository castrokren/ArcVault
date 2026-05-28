# Phase 18 — Installer Build Guide

## Overview

The Phase 18 implementation includes source files (`.nsi`, `.swift`, shell scripts) that must be **compiled** into distributable installer packages. This guide explains the build process for developers and CI/CD.

## What Users See vs. What Developers Build

### ✅ What End Users Get (Distribution)
- `ArcVault-Setup-1.1.0-windows-amd64.exe` — Double-click to install (no compilation needed)
- `ArcVault-Setup-1.1.0-macos-arm64.pkg` — Mount and follow installer (no compilation needed)
- `arcvault_1.1.0_amd64.deb` — `apt install` (no compilation needed)

### 🔧 What Developers/CI Must Do (Building)
- Compile `.nsi` → `.exe` using NSIS toolchain
- Compile `.swift` → macOS app bundle using Xcode tools
- Package `.deb` and `.rpm` using goreleaser's nfpm
- All automated in goreleaser's build pipeline

---

## Windows Installer Build

### Source Files
- `installer/windows/arcvault.nsi` — NSIS script (source code)
- `installer/windows/installer.ico` — App icon (optional)
- `installer/windows/welcome.bmp` — MUI2 sidebar graphic (optional)

### Build Requirements
- **NSIS 3.x** — Must be installed and in PATH
  - Download: https://nsis.sourceforge.io/Download
  - Windows installer or portable version

### Build Command
```bash
# Manual build (for testing)
makensis /V4 installer/windows/arcvault.nsi

# Output: ArcVault-Setup-1.1.0-windows-amd64.exe
```

### Goreleaser Integration
```yaml
after_hooks:
  - cmd: makensis /V4 /D__OUTFILE__={{ .Path }} installer/windows/arcvault.nsi
    dir: .
```

When `goreleaser release` runs, it:
1. Builds coordinator + agent + setup binaries
2. Runs `makensis` to compile `.nsi` → `.exe`
3. Includes `.exe` in GitHub release assets

### Developer Workflow
```bash
# 1. Install NSIS locally
choco install nsis  # or download from website

# 2. Test installer locally
makensis /V4 installer/windows/arcvault.nsi
# Creates: ArcVault-Setup-*.exe in current directory

# 3. Test the .exe installer
./ArcVault-Setup-1.1.0-windows-amd64.exe
# Double-click opens the Modern UI wizard
```

---

## macOS Installer Build

### Source Files
- `installer/macos/distribution.xml` — Component layout
- `installer/macos/welcome.html` — Welcome page
- `installer/macos/license.html` — License page
- `installer/macos/postinstall` — Post-install script
- `installer/macos/ArcVaultSetup.swift` — SwiftUI config wizard app

### Build Requirements
- **Xcode Command Line Tools** (macOS 10.13+)
  ```bash
  xcode-select --install
  ```
- **pkgbuild** and **productbuild** tools (included with Xcode)

### Build Process

**Step 1: Compile SwiftUI app**
```bash
swiftc installer/macos/ArcVaultSetup.swift -o ArcVaultSetup
# Creates: ArcVaultSetup binary
```

**Step 2: Create component packages**
```bash
# Coordinator package
pkgbuild --root . \
  --identifier com.arcvault.coordinator \
  --version 1.1.0 \
  --install-location /Applications/ArcVault \
  --scripts installer/macos \
  coordinator.pkg

# Agent package (similar)
pkgbuild --root . \
  --identifier com.arcvault.agent \
  --version 1.1.0 \
  --install-location /Applications/ArcVault \
  --scripts installer/macos \
  agent.pkg
```

**Step 3: Create distribution package**
```bash
productbuild --distribution installer/macos/distribution.xml \
  --resources installer/macos \
  ArcVault-Setup-1.1.0-macos-amd64.pkg
```

### Goreleaser Integration
```yaml
after_hooks:
  - cmd: ./scripts/build-macos-installer.sh {{ .Version }}
    dir: .
```

Create `scripts/build-macos-installer.sh`:
```bash
#!/bin/bash
set -e
VERSION=$1

# Compile Swift app
swiftc installer/macos/ArcVaultSetup.swift -o ArcVaultSetup

# Build component packages
pkgbuild --root . --identifier com.arcvault.coordinator \
  --version $VERSION --install-location /Applications/ArcVault \
  --scripts installer/macos coordinator.pkg

pkgbuild --root . --identifier com.arcvault.agent \
  --version $VERSION --install-location /Applications/ArcVault \
  --scripts installer/macos agent.pkg

# Create distribution
productbuild --distribution installer/macos/distribution.xml \
  --resources installer/macos \
  "ArcVault-Setup-${VERSION}-macos-amd64.pkg"

# Sign and notarize (future phase)
```

### Developer Workflow
```bash
# 1. Build locally (on macOS)
./scripts/build-macos-installer.sh 1.1.0

# 2. Test the .pkg installer
open ArcVault-Setup-1.1.0-macos-amd64.pkg
# Native Installer.app opens with wizard

# 3. Verify services
launchctl list | grep arcvault
```

---

## Linux Installer Build

### Source Files
- `installer/linux/postinst` — Post-install script
- `installer/linux/prerm` — Pre-remove script

### Build Requirements
- **goreleaser** with nfpm
  - Already configured in `.goreleaser.yaml`

### Build Process
Goreleaser nfpm automatically:
1. Copies binaries (coordinator, agent, setup)
2. Runs postinst after installation
3. Runs prerm before uninstallation
4. Generates .deb and .rpm packages

### Goreleaser Configuration
```yaml
nfpm:
  - id: arcvault-deb
    formats: [deb]
    scripts:
      postinstall: installer/linux/postinst
      preremove: installer/linux/prerm
    bindir: /usr/local/bin

  - id: arcvault-rpm
    formats: [rpm]
    scripts:
      postinstall: installer/linux/postinst
      preremove: installer/linux/prerm
```

### Build Command
```bash
# Goreleaser handles everything
goreleaser release --snapshot --rm-dist

# Output:
#   dist/arcvault_1.1.0_amd64.deb
#   dist/arcvault_1.1.0_x86_64.rpm
```

### Developer Workflow
```bash
# 1. Install goreleaser (if not already)
go install github.com/goreleaser/goreleaser@latest

# 2. Build locally
goreleaser build --snapshot --rm-dist

# 3. Test .deb installation
sudo apt install ./dist/arcvault_1.1.0_amd64.deb

# 4. Verify services
systemctl status arcvault-coordinator
```

---

## Complete CI/CD Release Workflow

### GitHub Actions Example
```yaml
name: Build Installers

on:
  push:
    tags:
      - 'v*'

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: choco install nsis
      - run: goreleaser release

  build-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: goreleaser release

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: goreleaser release
```

---

## Summary: What's a File, What's a Build Artifact

| File | Type | Built By | Distributed As |
|------|------|----------|-----------------|
| `installer/windows/arcvault.nsi` | Source code | Developer | Not distributed |
| `ArcVault-Setup-1.1.0-windows-amd64.exe` | Compiled artifact | makensis | ✅ Users download |
| `installer/macos/ArcVaultSetup.swift` | Source code | Developer | Not distributed |
| `ArcVault-Setup-1.1.0-macos-amd64.pkg` | Compiled artifact | productbuild | ✅ Users download |
| `installer/linux/postinst` | Shell script | goreleaser | Inside .deb/.rpm |
| `arcvault_1.1.0_amd64.deb` | Compiled artifact | nfpm | ✅ Users install |

---

## .NSI File Explanation

The `.nsi` file is **NSIS script source code**. NSIS (Nullsoft Scriptable Install System) is a tool for building Windows installers.

**Why you see "how to open" on Windows:**
- Windows doesn't recognize `.nsi` as an executable or document
- It's a build script, not a program
- Needs `makensis` compiler to convert script → `.exe`

**How it becomes a Windows installer:**
```
arcvault.nsi → [makensis compiler] → ArcVault-Setup-1.1.0-windows-amd64.exe
```

The `.exe` is what users download and double-click. The `.nsi` is just the source.

---

## Next Steps

1. **Update .goreleaser.yaml** with platform-specific build hooks
2. **Create build scripts** for macOS and Windows
3. **Test locally** on each platform
4. **Push to main** when installers build successfully
5. **Release v1.1.0** with all 5 installer artifacts

---

**Phase 18 Build Status:**
- ✅ Source files implemented
- ⏳ Build integration (goreleaser hooks)
- ⏳ Platform-specific testing
- ⏳ Release automation
