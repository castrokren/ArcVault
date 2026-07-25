# Sandbox Environment Setup Guide

Build a fully functional sandbox to compile ArcVault Phase 18 installers on Linux/Windows.

---

## Problem Summary

Current sandbox is missing:
- ❌ Go compiler
- ❌ NSIS toolchain (Windows .exe)
- ❌ Build tools (make, gcc, etc.)
- ❌ Package managers with sudo access

---

## Solution: Multi-Approach Setup

Choose the approach that works for your environment:

### **Approach 1: Use Docker (RECOMMENDED)**

Docker provides a complete isolated environment with all tools pre-installed.

#### Install Docker

**Windows/macOS:**
- Download: https://www.docker.com/products/docker-desktop
- Install normally (includes Docker CLI)

**Linux:**
```bash
# If you have sudo access
sudo apt-get install -y docker.io
sudo usermod -aG docker $USER

# If no sudo, try Podman (drop-in Docker replacement)
apt-get install -y podman
```

#### Create Dockerfile for ArcVault Build Environment

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    golang-go \
    git \
    nsis \
    wine \
    make \
    gcc \
    fpm \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Set up Go
ENV PATH="/usr/lib/go/bin:$PATH"

WORKDIR /arcvault
```

#### Build and Run Container

```bash
# Create Dockerfile in project root
cat > Dockerfile << 'EOF'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    golang-go git nsis wine make gcc fpm curl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /arcvault
EOF

# Build image
docker build -t arcvault-builder .

# Run container with project mounted
docker run -it -v $(pwd):/arcvault arcvault-builder bash

# Inside container:
cd /arcvault
./scripts/build-installers.sh 1.1.0
```

**Result:** All tools available in isolated environment. Clean up by exiting the container.

---

### **Approach 2: Download Pre-Built Binaries (Fastest)**

Don't compile—download pre-built tools directly.

#### Go Compiler

```bash
# Download latest Go
VERSION="1.25.0"
wget https://go.dev/dl/go${VERSION}.linux-amd64.tar.gz
tar -xzf go${VERSION}.linux-amd64.tar.gz -C /tmp

# Add to PATH
export PATH="/tmp/go/bin:$PATH"
go version  # Verify
```

#### NSIS for Windows .exe

```bash
# Download NSIS portable
wget https://sourceforge.net/projects/nsis/files/NSIS%203/3.09/nsis-3.09-portable.zip
unzip nsis-3.09-portable.zip -d /tmp/nsis

# Use with Wine on Linux (simulates Windows environment)
wine /tmp/nsis/makensis.exe installer/windows/arcvault.nsi
```

#### Goreleaser (Build Linux Packages)

```bash
# Download latest goreleaser
wget https://github.com/goreleaser/goreleaser/releases/download/v1.24.0/goreleaser_Linux_x86_64.tar.gz
tar -xzf goreleaser_Linux_x86_64.tar.gz -C /tmp

export PATH="/tmp/goreleaser:$PATH"
goreleaser --version  # Verify
```

#### FPM (Create .deb/.rpm)

```bash
# Download Ruby-based fpm
ruby_version=$(ruby --version 2>/dev/null || echo "missing")

# If Ruby available:
gem install --user-install fpm

# Otherwise, use pre-built container:
docker run -v $(pwd):/app jordansissel/fpm \
  -s dir -t deb -n arcvault ...
```

**Result:** All tools in /tmp, PATH configured. Tools persist during session.

---

### **Approach 3: System Package Installation (Most Compatible)**

If you have sudo access on Linux:

```bash
#!/bin/bash
# Install everything needed for Phase 18 builds

# Go
sudo apt-get update
sudo apt-get install -y golang-go

# NSIS (requires Wine for Linux)
sudo apt-get install -y nsis wine

# Build tools
sudo apt-get install -y \
    build-essential \
    make \
    gcc \
    git

# Goreleaser
curl -sL https://github.com/goreleaser/goreleaser/releases/download/v1.24.0/goreleaser_Linux_x86_64.tar.gz \
  | tar -xz -C /usr/local/bin

# FPM (package builder)
sudo apt-get install -y ruby ruby-dev
sudo gem install fpm

# Verify all tools
go version
makensis -VERSION
fpm --version
goreleaser --version

echo "✓ All tools installed and ready!"
```

**Result:** System-wide tools available. Survives restarts.

---

### **Approach 4: Windows + WSL2 (Best for Windows Developers)**

Windows Subsystem for Linux 2 gives you Linux environment on Windows.

#### Install WSL2

```powershell
# Windows PowerShell (admin)
wsl --install

# Restart, then in WSL terminal:
sudo apt-get update
sudo apt-get install -y golang-go git nsis make gcc fpm

# Clone repo inside WSL
cd /mnt/c/Projects/ArcVault2.0
./scripts/build-installers.sh 1.1.0
```

**Result:** Linux tools available, integrates with Windows files via /mnt/c/

---

### **Approach 5: macOS + Xcode (For .pkg Building)**

Only way to build native .pkg installers:

```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install Go
brew install go

# Install goreleaser
brew install goreleaser

# Verify
go version
productbuild --version
goreleaser --version

# Build macOS installer
./scripts/build-installers.sh 1.1.0 macos
```

**Result:** Native macOS .pkg installers (only way to get them).

---

## Recommended Setup by Platform

### For Linux Developers
**Best option:** Docker (Approach 1)
```bash
docker build -t arcvault-builder .
docker run -it -v $(pwd):/arcvault arcvault-builder bash
./scripts/build-installers.sh 1.1.0
```

### For Windows Developers
**Best option:** WSL2 (Approach 4)
```powershell
wsl --install
# Inside WSL:
sudo apt-get install golang-go nsis goreleaser ruby ruby-dev
sudo gem install fpm
./scripts/build-installers.sh 1.1.0
```

### For macOS Developers
**Best option:** System packages (Approach 3)
```bash
brew install go nsis goreleaser fpm
./scripts/build-installers.sh 1.1.0
```

### For CI/CD (GitHub Actions)
Use Docker in CI pipeline:
```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    container: arcvault-builder:latest
    steps:
      - uses: actions/checkout@v3
      - run: ./scripts/build-installers.sh 1.1.0
      - uses: actions/upload-artifact@v3
        with:
          path: ArcVault-Setup-*
```

---

## Quick Start: Complete Setup (30 minutes)

Choose your OS:

### **Linux**

```bash
# 1. Install Go
mkdir -p ~/tools
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz -O ~/tools/go.tar.gz
tar -xzf ~/tools/go.tar.gz -C ~/tools
export PATH="~/tools/go/bin:$PATH"

# 2. Install goreleaser
wget https://github.com/goreleaser/goreleaser/releases/download/v1.24.0/goreleaser_Linux_x86_64.tar.gz -O ~/tools/goreleaser.tar.gz
tar -xzf ~/tools/goreleaser.tar.gz -C ~/tools
export PATH="~/tools:$PATH"

# 3. Install NSIS (via Docker, since Wine + NSIS is complex)
docker pull madebuild/nsis
docker run -v $(pwd):/workspace madebuild/nsis \
  makensis /workspace/installer/windows/arcvault.nsi

# 4. Clone repo
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# 5. Build
./scripts/build-installers.sh 1.1.0

# Result: ArcVault-Setup-*.exe, .pkg, .deb, .rpm
```

### **Windows (using WSL2)**

```powershell
# PowerShell (admin)
wsl --install
# Restart computer

# In WSL terminal:
sudo apt-get update
sudo apt-get install -y golang-go git nsis
curl -sL https://github.com/goreleaser/goreleaser/releases/download/v1.24.0/goreleaser_Linux_x86_64.tar.gz | tar -xz -C /tmp
export PATH="/tmp:$PATH"

cd /mnt/c/Projects/ArcVault2.0
./scripts/build-installers.sh 1.1.0
```

### **macOS**

```bash
# Install Homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install tools
brew install go nsis goreleaser

# Clone repo
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# Build
./scripts/build-installers.sh 1.1.0
```

---

## Verify Installation

After setup, verify all tools are available:

```bash
# Go
go version
# Expected: go version go1.25.0 linux/amd64

# Goreleaser
goreleaser --version
# Expected: goreleaser version 1.24.0

# NSIS (if on Windows/WSL or Docker)
makensis -VERSION
# Expected: NSIS v3.09

# Check PATH
echo $PATH
# Should include locations where tools are installed
```

---

## Troubleshooting

### "command not found: go"
- Make sure ~/tools/go/bin is in PATH: `export PATH="~/tools/go/bin:$PATH"`
- Verify installation: `ls ~/tools/go/bin/go`

### "makensis: command not found"
- Use Docker instead of trying to run NSIS on Linux directly
- Or install Wine + NSIS on Linux (complex)

### "goreleaser: command not found"
- Verify extraction: `ls ~/tools/goreleaser`
- Add to PATH: `export PATH="~/tools:$PATH"`

### Docker permission denied
- Add user to docker group: `sudo usermod -aG docker $USER`
- Logout and login again

### fpm not installing
- Try with gem: `sudo gem install fpm`
- Or use Docker: `docker run --rm -v $(pwd):/app jordansissel/fpm ...`

---

## Production Recommendation

For **production builds** (actual releases):

1. **Use GitHub Actions** with Docker container
2. **Each platform builds on native OS:**
   - Windows .exe: `runs-on: windows-latest`
   - macOS .pkg: `runs-on: macos-latest`
   - Linux .deb/.rpm: `runs-on: ubuntu-latest`
3. **Artifacts signed and notarized**
4. **Released to GitHub automatically**

See `ci-cd-workflow.yml` for complete GitHub Actions setup.

---

## Next Steps

1. **Choose your approach** (Docker recommended)
2. **Run setup** (30 minutes)
3. **Test build:** `./scripts/build-installers.sh 1.1.0`
4. **Push artifacts:** `git add ArcVault-Setup-* && git commit`
5. **Release:** Create GitHub release with all installers

You'll have fully functional, native installers for all three platforms! 🚀
