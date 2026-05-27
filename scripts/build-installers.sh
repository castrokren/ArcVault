#!/bin/bash
# Phase 18: Build native installers for all platforms
# Usage: ./scripts/build-installers.sh [version] [platform]
#
# Examples:
#   ./scripts/build-installers.sh 1.1.0          # Build all
#   ./scripts/build-installers.sh 1.1.0 windows  # Build Windows only
#   ./scripts/build-installers.sh 1.1.0 macos    # Build macOS only
#   ./scripts/build-installers.sh 1.1.0 linux    # Build Linux only

set -e

VERSION=${1:-1.1.0}
PLATFORM=${2:-all}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "═══════════════════════════════════════════════════════"
echo "ArcVault Phase 18 — Installer Builder"
echo "Version: $VERSION"
echo "Platform: $PLATFORM"
echo "═══════════════════════════════════════════════════════"
echo

cd "$PROJECT_ROOT"

# Build coordinator, agent, and setup binaries first
echo "📦 Building binaries..."
go build -o coordinator ./coordinator
go build -o agent ./agent
go build -o cmd/setup/arcvault-setup ./cmd/setup

echo "✓ Binaries built"
echo

# Windows Installer
if [[ "$PLATFORM" == "all" || "$PLATFORM" == "windows" ]]; then
  echo "🪟 Building Windows installer..."

  if ! command -v makensis &> /dev/null; then
    echo "❌ Error: makensis not found. Install NSIS from https://nsis.sourceforge.io/"
    echo "   Windows: choco install nsis"
    echo "   Or download and install manually"
    exit 1
  fi

  # Create temp directory for installer
  TEMP_WIN=$(mktemp -d)
  trap "rm -rf $TEMP_WIN" EXIT

  # Copy binaries to temp
  cp coordinator "$TEMP_WIN/"
  cp agent "$TEMP_WIN/"
  cp cmd/setup/arcvault-setup "$TEMP_WIN/"

  # Build installer
  makensis /V4 \
    /D"OUTFILE=$PROJECT_ROOT/ArcVault-Setup-${VERSION}-windows-amd64.exe" \
    installer/windows/arcvault.nsi

  echo "✓ Windows installer: ArcVault-Setup-${VERSION}-windows-amd64.exe"
  echo
fi

# macOS Installer
if [[ "$PLATFORM" == "all" || "$PLATFORM" == "macos" ]]; then
  echo "🍎 Building macOS installer..."

  if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "⚠️  Warning: Building on non-macOS system. Skipping macOS build."
    echo "   macOS installer must be built on macOS with Xcode tools."
    echo "   Run this script on a Mac to build the .pkg installer."
    echo
  else
    if ! command -v productbuild &> /dev/null; then
      echo "❌ Error: productbuild not found. Install Xcode Command Line Tools:"
      echo "   xcode-select --install"
      exit 1
    fi

    # Create temp directory for installer
    TEMP_MAC=$(mktemp -d)
    trap "rm -rf $TEMP_MAC" EXIT

    # Create component packages
    echo "  Building component packages..."
    pkgbuild --root . \
      --identifier com.arcvault.coordinator \
      --version "$VERSION" \
      --install-location /Applications/ArcVault \
      --scripts installer/macos \
      "$TEMP_MAC/coordinator.pkg"

    pkgbuild --root . \
      --identifier com.arcvault.agent \
      --version "$VERSION" \
      --install-location /Applications/ArcVault \
      --scripts installer/macos \
      "$TEMP_MAC/agent.pkg"

    # Create distribution package
    echo "  Building distribution package..."
    productbuild \
      --distribution installer/macos/distribution.xml \
      --resources installer/macos \
      --package-path "$TEMP_MAC" \
      "$PROJECT_ROOT/ArcVault-Setup-${VERSION}-macos-amd64.pkg"

    # Also build arm64 variant
    productbuild \
      --distribution installer/macos/distribution.xml \
      --resources installer/macos \
      --package-path "$TEMP_MAC" \
      "$PROJECT_ROOT/ArcVault-Setup-${VERSION}-macos-arm64.pkg"

    echo "✓ macOS installers:"
    echo "  - ArcVault-Setup-${VERSION}-macos-amd64.pkg"
    echo "  - ArcVault-Setup-${VERSION}-macos-arm64.pkg"
    echo
  fi
fi

# Linux Installers (via goreleaser)
if [[ "$PLATFORM" == "all" || "$PLATFORM" == "linux" ]]; then
  echo "🐧 Building Linux installers..."

  if ! command -v goreleaser &> /dev/null; then
    echo "❌ Error: goreleaser not found. Install with:"
    echo "   go install github.com/goreleaser/goreleaser@latest"
    exit 1
  fi

  # Build snapshot (doesn't require git)
  goreleaser build \
    --snapshot \
    --rm-dist \
    --single-target \
    --skip-validate

  echo "✓ Linux installers created in ./dist/"
  ls -lh dist/arcvault_${VERSION}* 2>/dev/null || echo "   (Check dist/ directory)"
  echo
fi

echo "═══════════════════════════════════════════════════════"
echo "✅ Installer build complete!"
echo "═══════════════════════════════════════════════════════"
echo
echo "Generated artifacts:"
ls -lh "ArcVault-Setup-${VERSION}"-* 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
echo
echo "Next steps:"
echo "  1. Test installers on target platforms"
echo "  2. Verify services start and dashboard opens"
echo "  3. Push changes and create PR"
echo "  4. Release v${VERSION} with these artifacts"
