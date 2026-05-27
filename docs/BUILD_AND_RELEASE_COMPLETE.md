# Phase 18: Complete Build & Release Guide

**Status:** ✅ Implementation Complete | 🔧 Ready to Build | 📦 Ready to Release

---

## The Complete Picture

You now have a **production-ready Phase 18 implementation** with:

1. ✅ **Installer Source Code**
   - Windows NSIS script (`installer/windows/arcvault.nsi`)
   - macOS package files (`installer/macos/*`)
   - Linux post-install scripts (`installer/linux/*`)
   - Shared Go wizard (`cmd/setup/*`)

2. ✅ **Build Automation**
   - `scripts/build-installers.sh` — Local development builds
   - `.github/workflows/build-installers.yml` — CI/CD automation
   - `.goreleaser.yaml` — Release configuration

3. ✅ **Documentation**
   - `SANDBOX_SETUP_GUIDE.md` — Environment setup
   - `INSTALLER_BUILD_GUIDE.md` — Build process details
   - `PHASE_18_COMPLETION_GUIDE.md` — Implementation summary

---

## Path to Production Installers

### **Option 1: Build Locally (Recommended for Testing)**

**Time required:** 1-2 hours (setup + build)

```bash
# Step 1: Set up environment (30 mins)
# See SANDBOX_SETUP_GUIDE.md for your OS
# - Linux: Download Go, goreleaser, use Docker for NSIS
# - Windows: Use WSL2 or native tools
# - macOS: Use Homebrew

# Step 2: Build installers (30 mins)
cd /path/to/ArcVault
./scripts/build-installers.sh 1.1.0

# Result:
#   ArcVault-Setup-1.1.0-windows-amd64.exe
#   ArcVault-Setup-1.1.0-macos-amd64.pkg
#   ArcVault-Setup-1.1.0-macos-arm64.pkg
#   arcvault_1.1.0_amd64.deb
#   arcvault_1.1.0_x86_64.rpm

# Step 3: Test each installer on target platform
# Download and run on Windows, macOS, Linux
# Verify: Services start, dashboard opens, uninstall cleans

# Step 4: Commit and push
git add installer/ cmd/setup/ scripts/ .goreleaser.yaml *.md .github/
git commit -m "Phase 18: Complete native installer implementation"
git push origin feature/phase-18-installers

# Step 5: Create PR
gh pr create --title "Phase 18: Native GUI installers for all platforms" \
  --body "Closes #Phase18"
```

### **Option 2: Use GitHub Actions (Recommended for Release)**

**Time required:** 10 minutes (setup) + automated build

```bash
# Step 1: Create release tag
git tag -a v1.1.0 -m "Phase 18: Release 1.1.0 with native installers"

# Step 2: Push tag
git push origin v1.1.0

# Step 3: GitHub Actions automatically:
# - Builds Windows .exe on windows-latest
# - Builds macOS .pkg on macos-latest  
# - Builds Linux .deb/.rpm on ubuntu-latest
# - Creates GitHub release with all installers
# - (Requires signing/notarization credentials for production)

# Step 4: Monitor build
# Go to Actions tab → watch workflows complete

# Step 5: Download artifacts
# GitHub release page shows all 5 installer files
# Ready for production distribution
```

---

## Quick Start: 5 Minute Setup

### **Fastest Path (Using Docker)**

```bash
# 1. Install Docker
# - macOS/Windows: Download Docker Desktop
# - Linux: sudo apt-get install docker.io

# 2. Build image
cd /path/to/ArcVault
docker build -t arcvault-builder .

# 3. Run build inside container
docker run -it -v $(pwd):/arcvault arcvault-builder bash

# Inside container:
cd /arcvault
./scripts/build-installers.sh 1.1.0

# 4. Installers appear in /arcvault directory
# Exit container and test them
```

**Total time:** 5 minutes for setup + 30 minutes for build = 35 minutes to finished installers

---

## File Manifest: What You Have

```
ArcVault/
├── installer/
│   ├── windows/
│   │   └── arcvault.nsi              ← Source: compile with makensis
│   ├── macos/
│   │   ├── distribution.xml          ← Source: compile with productbuild
│   │   ├── welcome.html
│   │   ├── license.html
│   │   ├── postinstall               ← Run after pkg installation
│   │   └── ArcVaultSetup.swift       ← Compile: swiftc
│   └── linux/
│       ├── postinst                  ← Run after .deb/.rpm install
│       └── prerm                     ← Run before uninstall
│
├── cmd/setup/
│   ├── main.go                       ← Go: compile to binary
│   ├── wizard.go                     ← Shared cross-platform wizard
│   └── browser.go                    ← Cross-platform browser opener
│
├── scripts/
│   └── build-installers.sh           ← Orchestrates full build
│
├── .github/workflows/
│   └── build-installers.yml          ← GitHub Actions CI/CD
│
├── .goreleaser.yaml                  ← Release configuration
│
└── Documentation/
    ├── SANDBOX_SETUP_GUIDE.md        ← Environment setup
    ├── INSTALLER_BUILD_GUIDE.md      ← Build process details
    ├── PHASE_18_COMPLETION_GUIDE.md  ← Implementation summary
    └── BUILD_AND_RELEASE_COMPLETE.md ← This file
```

---

## Platform-Specific Requirements

| Platform | Requirements | Build Time | Output |
|----------|--------------|-----------|--------|
| **Windows** | Go + NSIS + makensis | 10 min | `ArcVault-Setup-*.exe` |
| **macOS** | Go + Xcode tools | 15 min | `ArcVault-Setup-*.pkg` |
| **Linux** | Go + fpm + nfpm | 10 min | `arcvault_*.deb`, `.rpm` |

**Best practice:** Use GitHub Actions to build each on its native OS runner.

---

## Testing Checklist

Before releasing to production:

### **Windows**
- [ ] Download `.exe` file
- [ ] Double-click installer
- [ ] Walk through Modern UI wizard
- [ ] Complete setup with coordinator or agent
- [ ] Services run: `sc query arcvault-coordinator`
- [ ] Dashboard opens in browser
- [ ] Uninstall from Add/Remove Programs
- [ ] Services stopped, files removed

### **macOS** (Intel)
- [ ] Download `.pkg` file
- [ ] Mount and follow Installer.app
- [ ] Complete SwiftUI config wizard
- [ ] Services running: `launchctl list | grep arcvault`
- [ ] Dashboard accessible: `open http://localhost:8080`
- [ ] Uninstall via manual file removal
- [ ] Services stopped, config removed

### **macOS** (Apple Silicon)
- [ ] Same as Intel, but with ARM64 `.pkg` file
- [ ] Verify native execution (not Rosetta)

### **Linux** (Debian/Ubuntu)
- [ ] `apt install ./arcvault_*.deb`
- [ ] Setup wizard runs automatically
- [ ] Services enabled: `systemctl is-enabled arcvault-coordinator`
- [ ] Dashboard accessible
- [ ] `apt remove arcvault` removes cleanly
- [ ] Services stopped, config preserved

### **Linux** (Red Hat/Fedora)
- [ ] `rpm -i arcvault-*.rpm`
- [ ] Setup wizard runs automatically
- [ ] Services running: `systemctl status arcvault-agent`
- [ ] Dashboard accessible
- [ ] `rpm -e arcvault` removes cleanly

---

## Release Checklist

Before pushing to production:

### **Code**
- [ ] All source files committed (`installer/`, `cmd/setup/`, scripts)
- [ ] Tests passing (if applicable)
- [ ] Documentation updated
- [ ] Feature branch reviewed and approved

### **Building**
- [ ] `.github/workflows/build-installers.yml` configured
- [ ] Goreleaser configuration tested locally
- [ ] Build scripts tested on each platform (or via Docker)
- [ ] All 5 artifacts generated successfully

### **Testing**
- [ ] Each installer tested on target OS
- [ ] Services start, dashboard opens
- [ ] Uninstall removes cleanly
- [ ] No errors in logs

### **Release**
- [ ] Version bumped to v1.1.0
- [ ] Tag created: `git tag -a v1.1.0 -m "..."`
- [ ] GitHub Actions builds automatically
- [ ] Release page shows all 5 installers
- [ ] Release notes include install instructions

### **Production**
- [ ] Posted to website/download page
- [ ] Announced in changelog
- [ ] Documented in wiki/guides
- [ ] Available for download

---

## Next Actions

### **Immediate (This Week)**
1. ✅ Choose your build approach (local or GitHub Actions)
2. ✅ Set up environment (follow SANDBOX_SETUP_GUIDE.md)
3. ✅ Build all installers
4. ✅ Test on each platform

### **Short Term (Next Week)**
1. ✅ Commit Phase 18 to feature branch
2. ✅ Create PR for code review
3. ✅ Merge to main after approval
4. ✅ Tag v1.1.0 release

### **Medium Term (Before Release)**
1. ✅ Configure code signing (Authenticode for Windows, Notarization for macOS)
2. ✅ Set up release notifications
3. ✅ Create user documentation
4. ✅ Prepare release notes

### **Long Term (Post Release)**
1. ✅ Monitor installer feedback
2. ✅ Plan Phase 19 (auto-update, enterprise features)
3. ✅ Gather usage metrics

---

## Support & Troubleshooting

### Can't find Go compiler?
See **SANDBOX_SETUP_GUIDE.md** → Approach 1 (Docker) for guaranteed working environment

### NSIS build failing?
- Ensure `makensis` is in PATH: `which makensis`
- On Linux, use Docker: `docker run madebuild/nsis makensis ...`
- Windows: Install NSIS from https://nsis.sourceforge.io

### macOS .pkg not building?
- Must be built on macOS (productbuild is macOS-only)
- Use GitHub Actions with `macos-latest` runner

### Linux .deb/.rpm issues?
- Ensure fpm is installed: `gem install fpm`
- Check permissions on scripts: `chmod +x installer/linux/*`

---

## Success Criteria

✅ You have successfully completed Phase 18 when:

1. **All installers built:** `.exe`, `.pkg`, `.deb`, `.rpm` created
2. **All installers tested:** Each installs and runs on its platform
3. **Services working:** Coordinator/Agent services start automatically
4. **User experience smooth:** Dashboard opens, setup wizard works, uninstall clean
5. **Code committed:** All source files in repo with documentation
6. **Released:** v1.1.0 tagged and published with all installers

---

## Celebrating Success 🎉

When all 5 installers are built and working:

```
✅ Windows .exe    — Modern UI, user-friendly
✅ macOS .pkg      — Native installer, SwiftUI wizard
✅ Linux .deb      — Package manager integration, systemd
✅ Linux .rpm      — RPM compatibility, systemd
✅ All binaries    — Signed, notarized, ready for production
```

**You have successfully delivered Phase 18 and are ready for v1.1.0 production release!**

---

**Phase 18 Implementation Status:**
- ✅ Design & Architecture Complete
- ✅ Installer Source Code Complete
- ✅ Build Scripts Complete
- ✅ CI/CD Automation Complete
- ⏳ Local Testing (Your next step)
- ⏳ Production Release (Final step)

**Next step:** Choose your build approach above and start building! 🚀
