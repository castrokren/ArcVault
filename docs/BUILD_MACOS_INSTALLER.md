# Build macOS Installers (.pkg)

Complete step-by-step guide to build both:
- `ArcVault-Setup-1.1.0-macos-amd64.pkg` (Intel Macs)
- `ArcVault-Setup-1.1.0-macos-arm64.pkg` (Apple Silicon)

**Time required:** 60 minutes  
**Difficulty:** Advanced  
**Platform:** macOS 10.13+ (Xcode required)
**Requirements:** Must be built ON macOS (productbuild is macOS-only)

---

## Prerequisites

### Install Xcode Command Line Tools
```bash
# Install (if not already present)
xcode-select --install

# Verify installation
xcode-select --print-path
# Expected: /Applications/Xcode.app/Contents/Developer
# or: /Library/Developer/CommandLineTools

# Verify productbuild is available
which productbuild
# Expected: /usr/bin/productbuild
```

### Install Homebrew (Package Manager)
```bash
# If not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go
brew install go

# Verify
go version
# Expected: go version go1.25.0 darwin/amd64 (or darwin/arm64)
```

---

## Build Steps

### Step 1: Clone Repository
```bash
cd ~/projects
git clone https://github.com/castrokren/ArcVault.git
cd ArcVault
```

### Step 2: Verify Project Structure
```bash
# Check that required files exist
test -f installer/macos/distribution.xml && echo "✓ distribution.xml found"
test -f installer/macos/postinstall && echo "✓ postinstall found"
test -f installer/macos/ArcVaultSetup.swift && echo "✓ ArcVaultSetup.swift found"
test -f cmd/setup/main.go && echo "✓ cmd/setup found"
```

### Step 3: Build Go Binaries (Both Architectures)

**For Intel (amd64):**
```bash
# Build coordinator
GOARCH=amd64 go build -o coordinator-amd64 ./coordinator
echo "✓ Built: coordinator-amd64"

# Build agent
GOARCH=amd64 go build -o agent-amd64 ./agent
echo "✓ Built: agent-amd64"

# Build setup wizard
GOARCH=amd64 go build -o cmd/setup/arcvault-setup-amd64 ./cmd/setup
echo "✓ Built: arcvault-setup-amd64"
```

**For Apple Silicon (arm64):**
```bash
# Build coordinator
GOARCH=arm64 go build -o coordinator-arm64 ./coordinator
echo "✓ Built: coordinator-arm64"

# Build agent
GOARCH=arm64 go build -o agent-arm64 ./agent
echo "✓ Built: agent-arm64"

# Build setup wizard
GOARCH=arm64 go build -o cmd/setup/arcvault-setup-arm64 ./cmd/setup
echo "✓ Built: arcvault-setup-arm64"
```

### Step 4: Create Temporary Build Directory
```bash
# Create temp directory for packaging
TEMP_BUILD=$(mktemp -d)
echo "Build temp: $TEMP_BUILD"

# Copy binaries for this architecture
# (Example for amd64, repeat for arm64)
ARCH="amd64"
mkdir -p "$TEMP_BUILD/$ARCH/roots"
mkdir -p "$TEMP_BUILD/$ARCH/scripts"

# Copy binaries
cp coordinator-$ARCH "$TEMP_BUILD/$ARCH/roots/"
cp agent-$ARCH "$TEMP_BUILD/$ARCH/roots/"
cp cmd/setup/arcvault-setup-$ARCH "$TEMP_BUILD/$ARCH/roots/"

# Copy scripts
cp installer/macos/postinstall "$TEMP_BUILD/$ARCH/scripts/"
chmod +x "$TEMP_BUILD/$ARCH/scripts/postinstall"

# Verify
ls -la "$TEMP_BUILD/$ARCH/"
```

### Step 5: Build Component Packages (pkgbuild)

**For Intel (amd64):**
```bash
ARCH="amd64"
VERSION="1.1.0"
TEMP_BUILD="/path/to/temp/from/above"

# Build Coordinator package
pkgbuild \
  --root "$TEMP_BUILD/$ARCH/roots" \
  --identifier com.arcvault.coordinator \
  --version "$VERSION" \
  --install-location /Applications/ArcVault \
  --scripts "$TEMP_BUILD/$ARCH/scripts" \
  "$TEMP_BUILD/$ARCH/coordinator.pkg"

echo "✓ Created: coordinator.pkg"

# Build Agent package
pkgbuild \
  --root "$TEMP_BUILD/$ARCH/roots" \
  --identifier com.arcvault.agent \
  --version "$VERSION" \
  --install-location /Applications/ArcVault \
  --scripts "$TEMP_BUILD/$ARCH/scripts" \
  "$TEMP_BUILD/$ARCH/agent.pkg"

echo "✓ Created: agent.pkg"

# Verify packages
ls -lh "$TEMP_BUILD/$ARCH/"*.pkg
```

**For Apple Silicon (arm64):**
```bash
# Same commands as above, replace $ARCH with "arm64"
ARCH="arm64"
VERSION="1.1.0"
TEMP_BUILD="/path/to/temp"

pkgbuild \
  --root "$TEMP_BUILD/$ARCH/roots" \
  --identifier com.arcvault.coordinator \
  --version "$VERSION" \
  --install-location /Applications/ArcVault \
  --scripts "$TEMP_BUILD/$ARCH/scripts" \
  "$TEMP_BUILD/$ARCH/coordinator.pkg"

pkgbuild \
  --root "$TEMP_BUILD/$ARCH/roots" \
  --identifier com.arcvault.agent \
  --version "$VERSION" \
  --install-location /Applications/ArcVault \
  --scripts "$TEMP_BUILD/$ARCH/scripts" \
  "$TEMP_BUILD/$ARCH/agent.pkg"
```

### Step 6: Create Distribution Package (productbuild)

**For Intel (amd64):**
```bash
ARCH="amd64"
VERSION="1.1.0"
TEMP_BUILD="/path/to/temp"

productbuild \
  --distribution installer/macos/distribution.xml \
  --resources installer/macos \
  --package-path "$TEMP_BUILD/$ARCH" \
  "ArcVault-Setup-${VERSION}-macos-${ARCH}.pkg"

echo "✓ Created: ArcVault-Setup-${VERSION}-macos-${ARCH}.pkg"
```

**For Apple Silicon (arm64):**
```bash
ARCH="arm64"
VERSION="1.1.0"
TEMP_BUILD="/path/to/temp"

productbuild \
  --distribution installer/macos/distribution.xml \
  --resources installer/macos \
  --package-path "$TEMP_BUILD/$ARCH" \
  "ArcVault-Setup-${VERSION}-macos-${ARCH}.pkg"

echo "✓ Created: ArcVault-Setup-${VERSION}-macos-${ARCH}.pkg"
```

### Step 7: Verify Installers
```bash
# Check both .pkg files exist
ls -lh ArcVault-Setup-*.pkg

# Expected output:
# ArcVault-Setup-1.1.0-macos-amd64.pkg  (50-100 MB)
# ArcVault-Setup-1.1.0-macos-arm64.pkg  (50-100 MB)
```

---

## Testing the Installers

### Test 1: Install on Intel Mac
```bash
# 1. Double-click installer or mount it
open ArcVault-Setup-1.1.0-macos-amd64.pkg

# 2. Native Installer.app opens
# 3. Select components and installation location
# 4. SwiftUI config wizard appears
# 5. Complete setup
```

### Test 2: Install on Apple Silicon Mac
```bash
open ArcVault-Setup-1.1.0-macos-arm64.pkg

# Same process as Intel, but runs natively on ARM64
# (not emulated via Rosetta)
```

### Test 3: Verify Services
```bash
# Check if launchd services are registered
launchctl list | grep arcvault

# Expected output:
# - com.arcvault.coordinator
# - com.arcvault.agent (if agent installed)

# Check service status
launchctl status com.arcvault.coordinator
# Expected: running

# View service logs
log stream --predicate 'process == "coordinator"'
```

### Test 4: Test Dashboard Access
```bash
# Open dashboard (should auto-open, but try manually)
open http://localhost:8080

# Or check if port is listening
lsof -i :8080
```

### Test 5: Test Uninstall
```bash
# Stop services
launchctl stop com.arcvault.coordinator
launchctl stop com.arcvault.agent

# Disable services
launchctl disable system/com.arcvault.coordinator
launchctl disable system/com.arcvault.agent

# Remove plist files
sudo rm /Library/LaunchDaemons/com.arcvault.coordinator.plist
sudo rm /Library/LaunchDaemons/com.arcvault.agent.plist

# Remove application files
sudo rm -rf /Applications/ArcVault

# Remove config
rm -rf ~/.arcvault

# Verify removal
launchctl list | grep arcvault
# Should return nothing
```

---

## Troubleshooting

### Error: "productbuild: command not found"
**Solution:**
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Or full Xcode
xcode-select --switch /Applications/Xcode.app/Contents/Developer

# Verify
which productbuild
```

### Error: "Invalid architecture"
**Solution:**
```bash
# Make sure you're building for correct architecture
uname -m
# Should show: arm64 (Apple Silicon) or x86_64 (Intel)

# Build for your current architecture:
go build ./coordinator
# Don't use GOARCH flag - it will auto-detect
```

### Error: "Code signature required"
**Solution (Future Phase - Code Signing):**
```bash
# For production releases, sign the .pkg:
productsign --sign "Developer ID Installer" \
  "ArcVault-Setup-1.1.0-macos-amd64.pkg" \
  "ArcVault-Setup-1.1.0-macos-amd64-signed.pkg"

# Requires Apple Developer Certificate (future phase)
```

### SwiftUI Wizard Not Launching
**Solution:**
```bash
# Compile the Swift app separately
swiftc installer/macos/ArcVaultSetup.swift \
  -o cmd/setup/ArcVaultSetup

# Or ensure it's bundled in the postinstall script
cat installer/macos/postinstall | grep ArcVaultSetup
```

---

## Advanced: Code Signing & Notarization

**Note:** This is for production releases. Development/testing doesn't require signing.

```bash
# Sign the .pkg (requires Apple Developer ID)
productsign --sign "Developer ID Installer: Company Name" \
  "ArcVault-Setup-1.1.0-macos-amd64.pkg" \
  "ArcVault-Setup-1.1.0-macos-amd64-signed.pkg"

# Notarize with Apple (required for Big Sur+)
xcrun notarytool submit "ArcVault-Setup-1.1.0-macos-amd64-signed.pkg" \
  --apple-id "your-apple-id@example.com" \
  --password "app-specific-password" \
  --team-id "XXXXXXXXXX"

# Wait for approval, then staple
xcrun stapler staple "ArcVault-Setup-1.1.0-macos-amd64-signed.pkg"
```

---

## Success Criteria

✅ **You've successfully built macOS installers when:**

1. **Files exist:**
   - `ArcVault-Setup-1.1.0-macos-amd64.pkg` (50-100 MB)
   - `ArcVault-Setup-1.1.0-macos-arm64.pkg` (50-100 MB)

2. **Can double-click .pkg:**
   - Native Installer.app opens
   - Component selection shows

3. **Installation completes:**
   - No errors during install
   - Postinstall script runs

4. **SwiftUI wizard launches:**
   - Config form appears
   - Setup completes

5. **Services register:**
   - `launchctl list | grep arcvault` shows services
   - Services running: `launchctl status com.arcvault.coordinator`

6. **Dashboard accessible:**
   - Browser opens automatically to `http://localhost:8080`
   - Or accessible manually

7. **Uninstall works:**
   - Services stop cleanly
   - Files removed
   - No leftover processes

---

## Next Steps

1. **Build on native macOS machine** (required - productbuild is macOS-only)
2. **Test on both Intel and Apple Silicon Macs** (if possible)
3. **Commit to git:**
   ```bash
   git add installer/macos/ cmd/setup/ *.pkg
   git commit -m "Phase 18: macOS installers built and tested"
   git push
   ```
4. **Build Linux installers** (see BUILD_LINUX_INSTALLERS.md)

---

## Reference: Distribution.xml Details

The `installer/macos/distribution.xml` file:
- Defines installer UI flow
- Lists component packages (Coordinator, Agent)
- Sets install location (`/Applications/ArcVault`)
- Specifies welcome/license pages

Key sections:
```xml
<pkg-ref id="com.arcvault.coordinator" version="1.1.0">
  coordinator.pkg
</pkg-ref>

<choice id="com.arcvault.coordinator" title="Coordinator">
  <pkg-ref id="com.arcvault.coordinator"/>
</choice>
```

---

**macOS Installer Build Complete!** 🎉

Next: Build Linux (.deb and .rpm) installers
