# Build Linux Installers (.deb and .rpm)

Complete step-by-step guide to build:
- `arcvault_1.1.0_amd64.deb` (Debian/Ubuntu)
- `arcvault_1.1.0_x86_64.rpm` (Red Hat/Fedora/Rocky)

**Time required:** 45 minutes  
**Difficulty:** Intermediate  
**Platform:** Linux (Debian-based or RPM-based)
**Methods:** Using FPM or Goreleaser

---

## Prerequisites

### Method 1: Using FPM (Flexible Package Manager) - Recommended

**Install on Debian/Ubuntu:**
```bash
sudo apt-get update
sudo apt-get install -y ruby ruby-dev

# Install FPM via Ruby gems
sudo gem install fpm

# Verify
fpm --version
# Expected: 1.14.0 or newer
```

**Install on Red Hat/Fedora/Rocky:**
```bash
sudo dnf install -y ruby ruby-devel

# Install FPM
sudo gem install fpm

# Verify
fpm --version
```

### Method 2: Using Goreleaser (For CI/CD)

**Install Goreleaser:**
```bash
# Via package manager
sudo apt-get install -y goreleaser  # Debian/Ubuntu
# OR
sudo dnf install -y goreleaser      # Fedora/RHEL

# OR download binary
wget https://github.com/goreleaser/goreleaser/releases/download/v1.24.0/goreleaser_Linux_x86_64.tar.gz
tar -xzf goreleaser_Linux_x86_64.tar.gz -C ~/bin

# Verify
goreleaser --version
# Expected: 1.24.0 or newer
```

### Install Go Compiler
```bash
# On Ubuntu/Debian
sudo apt-get install -y golang-go

# On Fedora/RHEL
sudo dnf install -y golang

# Verify
go version
# Expected: go version go1.25.0 linux/amd64
```

---

## Build Steps (Using FPM - Easiest)

### Step 1: Clone Repository
```bash
cd ~/projects
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault
```

### Step 2: Verify Project Structure
```bash
# Check required files exist
test -f cmd/setup/main.go && echo "✓ cmd/setup found"
test -f installer/linux/postinst && echo "✓ postinst found"
test -f installer/linux/prerm && echo "✓ prerm found"
test -f coordinator/main.go && echo "✓ coordinator found"
test -f agent/main.go && echo "✓ agent found"
```

### Step 3: Build Go Binaries
```bash
# Create dist directory for binaries
mkdir -p dist

# Build coordinator
go build -o dist/coordinator ./coordinator
echo "✓ Built: coordinator"

# Build agent
go build -o dist/agent ./agent
echo "✓ Built: agent"

# Build setup wizard
go build -o dist/arcvault-setup ./cmd/setup
echo "✓ Built: arcvault-setup"

# Verify binaries
ls -lh dist/
```

### Step 4: Make Scripts Executable
```bash
# Set correct permissions
chmod +x installer/linux/postinst
chmod +x installer/linux/prerm

# Verify
ls -l installer/linux/
# Expected: -rwxr-xr-x
```

### Step 5: Build .deb Package (Debian/Ubuntu)
```bash
VERSION="1.1.0"
ARCH="amd64"

fpm -s dir -t deb \
  -n arcvault \
  -v $VERSION \
  --description "User-friendly backup orchestration system" \
  --maintainer "ArcVault Team <team@arcvault.io>" \
  --url "https://github.com/castrokren/ArcVault" \
  --vendor "ArcVault" \
  --architecture $ARCH \
  --after-install installer/linux/postinst \
  --before-remove installer/linux/prerm \
  --directories /usr/local/bin \
  dist/coordinator=/usr/local/bin/coordinator \
  dist/agent=/usr/local/bin/agent \
  dist/arcvault-setup=/usr/local/bin/arcvault-setup

echo "✓ Created: arcvault_${VERSION}_${ARCH}.deb"

# Verify package
ls -lh arcvault_*.deb
```

**Expected output:**
```
Created deb package {:path=>"arcvault_1.1.0_amd64.deb"}
```

### Step 6: Build .rpm Package (Red Hat/Fedora/Rocky)
```bash
VERSION="1.1.0"
ARCH="x86_64"

fpm -s dir -t rpm \
  -n arcvault \
  -v $VERSION \
  --description "User-friendly backup orchestration system" \
  --maintainer "ArcVault Team <team@arcvault.io>" \
  --url "https://github.com/castrokren/ArcVault" \
  --vendor "ArcVault" \
  --architecture $ARCH \
  --rpm-dist "1" \
  --after-install installer/linux/postinst \
  --before-remove installer/linux/prerm \
  dist/coordinator=/usr/local/bin/coordinator \
  dist/agent=/usr/local/bin/agent \
  dist/arcvault-setup=/usr/local/bin/arcvault-setup

echo "✓ Created: arcvault-${VERSION}-1.${ARCH}.rpm"

# Verify package
ls -lh arcvault-*.rpm
```

**Expected output:**
```
Created rpm package {:path=>"arcvault-1.1.0-1.x86_64.rpm"}
```

### Step 7: Verify Packages
```bash
# List all installers
ls -lh arcvault_*.deb arcvault-*.rpm

# Inspect .deb package
dpkg -c arcvault_1.1.0_amd64.deb | head -20

# Inspect .rpm package
rpm -qlp arcvault-1.1.0-1.x86_64.rpm | head -20

# Expected files in both:
# /usr/local/bin/coordinator
# /usr/local/bin/agent
# /usr/local/bin/arcvault-setup
```

---

## Alternative: Build Steps (Using Goreleaser)

### One-Command Build
```bash
cd ~/projects/ArcVault

# Build snapshot (no git tag required, doesn't publish)
goreleaser build --snapshot --rm-dist

# Build for release (requires git tag)
git tag -a v1.1.0 -m "Phase 18: Release 1.1.0"
goreleaser release --rm-dist
```

**Expected output:**
```
dist/
├── arcvault_1.1.0_amd64.deb
├── arcvault_1.1.0_x86_64.rpm
├── ...
```

Goreleaser uses the `.goreleaser.yaml` configuration automatically.

---

## Testing the Installers

### Test 1: Install .deb Package (Ubuntu/Debian)
```bash
# Install package
sudo apt install ./arcvault_1.1.0_amd64.deb

# Expected output:
# Setting up arcvault (1.1.0) ...
# (postinst script runs, setup wizard appears)
```

### Test 2: Install .rpm Package (Fedora/RHEL)
```bash
# Install package
sudo rpm -i arcvault-1.1.0-1.x86_64.rpm

# Expected output:
# (postinst script runs, setup wizard appears)
```

### Test 3: Verify Installation
```bash
# Check binaries are installed
which coordinator
which agent
which arcvault-setup

# Expected output:
# /usr/local/bin/coordinator
# /usr/local/bin/agent
# /usr/local/bin/arcvault-setup

# Run setup wizard manually (if not run during install)
arcvault-setup

# Or directly start coordinator
coordinator init
coordinator start
```

### Test 4: Verify Service Registration
```bash
# Check if systemd service is registered (setup wizard should do this)
systemctl list-units --type=service | grep arcvault

# Or check manually
ls -la /etc/systemd/system/arcvault*

# Start service
sudo systemctl start arcvault-coordinator

# Check status
systemctl status arcvault-coordinator

# View logs
journalctl -u arcvault-coordinator -n 20
```

### Test 5: Test Dashboard Access
```bash
# Check if coordinator is listening
ss -tlnp | grep 8080

# Or test connection
curl http://localhost:8080

# Open in browser
firefox http://localhost:8080 &
chromium http://localhost:8080 &
```

### Test 6: Test Uninstall

**For .deb:**
```bash
# Remove package
sudo apt remove arcvault

# Or complete purge (remove config too)
sudo apt purge arcvault

# Verify removal
which arcvault-setup  # Should not find it
ls ~/.arcvault        # Config removed
systemctl status arcvault-coordinator  # Should fail
```

**For .rpm:**
```bash
# Remove package
sudo rpm -e arcvault

# Verify removal
which arcvault-setup  # Should not find it
systemctl status arcvault-coordinator  # Should fail
```

---

## Troubleshooting

### Error: "fpm: command not found"
**Solution:**
```bash
# Install FPM
sudo gem install fpm

# Or check if installed
gem list | grep fpm

# If gem fails, try via apt
sudo apt-get install -y fpm  # Some distros package it
```

### Error: "Failed to calculate target architecture"
**Solution:**
```bash
# FPM auto-detects architecture
# If it fails, specify explicitly:
fpm -a amd64 ...  # For 64-bit Intel
fpm -a arm64 ...  # For ARM (Raspberry Pi, etc.)

# Check your system architecture
uname -m
# Expected: x86_64 (Intel) or aarch64 (ARM)
```

### Error: "dpkg: error processing package"
**Solution:**
```bash
# This usually means duplicate installation
# Remove existing package first:
sudo apt remove -y arcvault

# Then reinstall:
sudo apt install ./arcvault_1.1.0_amd64.deb
```

### Postinstall Script Not Running
**Solution:**
```bash
# Check if script is executable
chmod +x installer/linux/postinst
chmod +x installer/linux/prerm

# Rebuild package
fpm -s dir -t deb \
  -n arcvault \
  -v 1.1.0 \
  --after-install installer/linux/postinst \
  --before-remove installer/linux/prerm \
  dist/coordinator=/usr/local/bin/coordinator \
  dist/agent=/usr/local/bin/agent \
  dist/arcvault-setup=/usr/local/bin/arcvault-setup

# Test script manually
bash installer/linux/postinst
```

---

## Advanced: Signing Packages

**Sign .deb package (requires GPG key):**
```bash
# Create GPG key if needed
gpg --gen-key

# Sign package
dpkg-sig --sign builder arcvault_1.1.0_amd64.deb

# Verify signature
dpkg-sig --verify arcvault_1.1.0_amd64.deb
```

**Sign .rpm package:**
```bash
# Add GPG key to RPM
rpm --import your-gpg-key.asc

# Sign package
rpm --addsign arcvault-1.1.0-1.x86_64.rpm

# Verify signature
rpm --checksig arcvault-1.1.0-1.x86_64.rpm
```

---

## Success Criteria

✅ **You've successfully built Linux installers when:**

1. **Files exist:**
   - `arcvault_1.1.0_amd64.deb` (20-50 MB)
   - `arcvault_1.1.0_x86_64.rpm` (20-50 MB)

2. **Can install .deb:**
   - `sudo apt install ./arcvault_*.deb` succeeds
   - Setup wizard runs automatically

3. **Can install .rpm:**
   - `sudo rpm -i arcvault-*.rpm` succeeds
   - Setup wizard runs automatically

4. **Binaries available:**
   - `which coordinator` finds binary
   - `which arcvault-setup` finds binary

5. **Services register:**
   - Systemd service file created: `/etc/systemd/system/arcvault-coordinator.service`
   - Service can start: `systemctl start arcvault-coordinator`

6. **Dashboard accessible:**
   - `curl http://localhost:8080` returns HTML
   - Browser can access dashboard

7. **Uninstall works:**
   - `apt remove` or `rpm -e` completes
   - Services stopped and disabled
   - Binaries removed from `/usr/local/bin`
   - Config preserved in `~/.arcvault` (optional)

---

## FPM Command Reference

**Common FPM options:**
```bash
fpm [options]
  -s dir            # Source type: directory (copy files)
  -t deb            # Target type: Debian package
  -t rpm            # Target type: RPM package
  -n NAME           # Package name
  -v VERSION        # Package version
  --description     # Short description
  --maintainer      # Maintainer email
  --after-install   # Script to run after installation
  --before-remove   # Script to run before uninstall
  path/to/files     # Files to include (multiple okay)
```

---

## Next Steps

1. **Build both .deb and .rpm packages:**
   ```bash
   # Build .deb
   fpm -s dir -t deb ... arcvault_1.1.0_amd64.deb
   
   # Build .rpm
   fpm -s dir -t rpm ... arcvault-1.1.0-1.x86_64.rpm
   ```

2. **Test on both Debian-based and RPM-based systems:**
   - Ubuntu/Debian for .deb
   - Fedora/RHEL/Rocky for .rpm

3. **Commit to git:**
   ```bash
   git add installer/linux/ dist/ *.deb *.rpm
   git commit -m "Phase 18: Linux installers built and tested"
   git push
   ```

4. **All three platforms complete!**
   - ✅ Windows .exe
   - ✅ macOS .pkg
   - ✅ Linux .deb/.rpm

---

**Linux Installer Build Complete!** 🎉

All three platform installers are now built. Next: Release v1.1.0!
