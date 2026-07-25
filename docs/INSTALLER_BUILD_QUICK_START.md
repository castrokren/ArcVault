# Installer Build Quick Start

Choose your OS and follow the step-by-step guide for that platform.

---

## 🪟 Windows (.exe)

**Requirements:**
- Windows 10+ or Windows 11
- Go 1.25.0+
- NSIS 3.09+
- Git

**Build Time:** 45 minutes

**Quick Steps:**
```powershell
# 1. Install Go, NSIS, Git
# 2. Clone repo
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# 3. Build binaries
go build -o coordinator.exe ./coordinator
go build -o agent.exe ./agent
go build -o cmd/setup/arcvault-setup.exe ./cmd/setup

# 4. Build installer
makensis /V4 installer/windows/arcvault.nsi

# 5. Rename output
ren ArcVault-Setup.exe ArcVault-Setup-1.1.0-windows-amd64.exe

# Result: ArcVault-Setup-1.1.0-windows-amd64.exe (5-10 MB)
```

**Detailed Guide:** See **[build-guides/BUILD_WINDOWS_INSTALLER.md](build-guides/BUILD_WINDOWS_INSTALLER.md)**

---

## 🍎 macOS (.pkg)

**Requirements:**
- macOS 10.13+ (Intel or Apple Silicon)
- Xcode Command Line Tools (required!)
- Go 1.25.0+
- Git
- Homebrew (optional)

**Build Time:** 60 minutes

**Quick Steps (Intel):**
```bash
# 1. Install Xcode tools
xcode-select --install

# 2. Install Go (via Homebrew)
brew install go

# 3. Clone repo
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# 4. Build binaries (Intel)
GOARCH=amd64 go build -o coordinator ./coordinator
GOARCH=amd64 go build -o agent ./agent
GOARCH=amd64 go build -o cmd/setup/arcvault-setup ./cmd/setup

# 5. Build component packages
pkgbuild --root . --identifier com.arcvault.coordinator \
  --version 1.1.0 --install-location /Applications/ArcVault \
  --scripts installer/macos coordinator.pkg

# 6. Build distribution package
productbuild --distribution installer/macos/distribution.xml \
  --resources installer/macos --package-path . \
  ArcVault-Setup-1.1.0-macos-amd64.pkg

# Result: ArcVault-Setup-1.1.0-macos-amd64.pkg (50-100 MB)
```

**Quick Steps (Apple Silicon):**
```bash
# Same as Intel, but replace amd64 with arm64:
GOARCH=arm64 go build -o coordinator ./coordinator
GOARCH=arm64 go build -o agent ./agent
GOARCH=arm64 go build -o cmd/setup/arcvault-setup ./cmd/setup

# Rest is same, produces ArcVault-Setup-1.1.0-macos-arm64.pkg
```

**Detailed Guide:** See **[build-guides/BUILD_MACOS_INSTALLER.md](build-guides/BUILD_MACOS_INSTALLER.md)**

---

## 🐧 Linux (.deb and .rpm)

**Requirements (Debian/Ubuntu):**
- Go 1.25.0+
- Ruby + FPM
- Git

**Requirements (Fedora/RHEL/Rocky):**
- Go 1.25.0+
- Ruby + FPM
- Git

**Build Time:** 45 minutes

**Quick Steps:**
```bash
# 1. Install dependencies
sudo apt-get install -y golang-go ruby ruby-dev
sudo gem install fpm

# 2. Clone repo
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault

# 3. Build binaries
mkdir -p dist
go build -o dist/coordinator ./coordinator
go build -o dist/agent ./agent
go build -o dist/arcvault-setup ./cmd/setup

# 4. Make scripts executable
chmod +x installer/linux/postinst installer/linux/prerm

# 5. Build .deb package
fpm -s dir -t deb -n arcvault -v 1.1.0 \
  --description "User-friendly backup orchestration system" \
  --maintainer "ArcVault Team" \
  --after-install installer/linux/postinst \
  --before-remove installer/linux/prerm \
  dist/coordinator=/usr/local/bin/coordinator \
  dist/agent=/usr/local/bin/agent \
  dist/arcvault-setup=/usr/local/bin/arcvault-setup

# 6. Build .rpm package
fpm -s dir -t rpm -n arcvault -v 1.1.0 \
  --description "User-friendly backup orchestration system" \
  --maintainer "ArcVault Team" \
  --after-install installer/linux/postinst \
  --before-remove installer/linux/prerm \
  dist/coordinator=/usr/local/bin/coordinator \
  dist/agent=/usr/local/bin/agent \
  dist/arcvault-setup=/usr/local/bin/arcvault-setup

# Results:
# arcvault_1.1.0_amd64.deb (20-50 MB)
# arcvault-1.1.0-1.x86_64.rpm (20-50 MB)
```

**Detailed Guide:** See **[build-guides/BUILD_LINUX_INSTALLERS.md](build-guides/BUILD_LINUX_INSTALLERS.md)**

---

## Testing Checklist (All Platforms)

After building, test each installer:

### Windows
```powershell
# 1. Double-click .exe
.\ArcVault-Setup-1.1.0-windows-amd64.exe

# 2. Walk through wizard
# 3. Verify service: sc query arcvault-coordinator
# 4. Test dashboard: http://localhost:8080
# 5. Uninstall and verify cleanup
```

### macOS
```bash
# 1. Open .pkg
open ArcVault-Setup-1.1.0-macos-amd64.pkg

# 2. Walk through Installer.app
# 3. Complete SwiftUI wizard
# 4. Verify services: launchctl list | grep arcvault
# 5. Test dashboard: open http://localhost:8080
# 6. Test uninstall
```

### Linux
```bash
# For .deb:
sudo apt install ./arcvault_1.1.0_amd64.deb

# For .rpm:
sudo rpm -i arcvault-1.1.0-1.x86_64.rpm

# Verify
which coordinator
arcvault-setup
systemctl status arcvault-coordinator

# Test uninstall
sudo apt remove arcvault  # or: sudo rpm -e arcvault
```

---

## All 5 Installers Summary

| OS | File | Build Command | Build Time |
|---|------|--------|-----------|
| **Windows** | ArcVault-Setup-1.1.0-windows-amd64.exe | makensis | 45 min |
| **macOS Intel** | ArcVault-Setup-1.1.0-macos-amd64.pkg | productbuild | 60 min |
| **macOS Silicon** | ArcVault-Setup-1.1.0-macos-arm64.pkg | productbuild | 60 min |
| **Linux Debian** | arcvault_1.1.0_amd64.deb | fpm | 45 min |
| **Linux RPM** | arcvault-1.1.0-1.x86_64.rpm | fpm | 45 min |

---

## Parallel Building (Recommended)

You don't need to build everything on one machine. Build each on its native OS:

```
Team Setup:
┌─ Developer 1 (Windows) → Builds .exe
├─ Developer 2 (macOS) → Builds .pkg (both Intel & Silicon)
└─ Developer 3 (Linux) → Builds .deb & .rpm
```

Or use GitHub Actions for completely automated builds:
```bash
# Just tag release, GitHub Actions builds all
git tag v1.1.0
git push origin v1.1.0
# ✓ Windows .exe built on windows-latest
# ✓ macOS .pkg built on macos-latest
# ✓ Linux .deb/.rpm built on ubuntu-latest
```

---

## Common Issues & Solutions

### "Go: command not found"
- Windows: Download & install from https://go.dev/dl/go1.25.0.windows-amd64.msi
- macOS: `brew install go`
- Linux: `sudo apt-get install golang-go`

### "makensis: command not found"
- Windows: Download from https://sourceforge.net/projects/nsis/
- Or via Chocolatey: `choco install nsis`

### "productbuild: command not found"
- macOS only: Must build on macOS
- Install Xcode: `xcode-select --install`

### "fpm: command not found"
- Linux: `sudo gem install fpm`

### Installer size larger than expected
- Check you're not including source files
- Use `strip` to reduce binary size: `strip dist/coordinator`

---

## Success = All 5 Installers Built

Once you have:
- ✅ ArcVault-Setup-1.1.0-windows-amd64.exe
- ✅ ArcVault-Setup-1.1.0-macos-amd64.pkg
- ✅ ArcVault-Setup-1.1.0-macos-arm64.pkg
- ✅ arcvault_1.1.0_amd64.deb
- ✅ arcvault-1.1.0-1.x86_64.rpm

You're ready to:
1. Commit to git
2. Create v1.1.0 release
3. Publish to GitHub
4. Ship to production! 🚀

---

## Next Steps

1. **Choose your platform(s)** — Windows, macOS, Linux, or all
2. **Follow the detailed guide** for your OS
3. **Build the installer(s)**
4. **Test on target platform**
5. **Commit and release**

---

**Total build time if building locally on one machine:** ~3.5 hours  
**Total build time if using separate machines + GitHub Actions:** ~30 minutes

Start now! Pick your OS above and follow the detailed guide. 🚀
