#!/bin/bash
set -euo pipefail

if command -v docker >/dev/null 2>&1; then
  echo "Docker CLI already installed. Skipping download."
  exit 0
fi

ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)
    DOCKER_URL="https://desktop.docker.com/mac/main/amd64/Docker.dmg"
    ;;
  arm64|aarch64)
    DOCKER_URL="https://desktop.docker.com/mac/main/arm64/Docker.dmg"
    ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

if [[ "${ARCV_ALLOW_SUDO:-1}" = "1" && "$EUID" -ne 0 ]]; then
  echo "Elevating privileges to install Docker Desktop..."
  exec sudo ARCV_ALLOW_SUDO=0 "$0" "$@"
fi

TMP_DIR="$(mktemp -d /tmp/arcvault_docker.XXXXXX)"
DMG_PATH="$TMP_DIR/Docker.dmg"
MOUNT_POINT="$TMP_DIR/mount"
mkdir -p "$MOUNT_POINT"

cleanup() {
  if mount | grep -q "$MOUNT_POINT"; then
    hdiutil detach "$MOUNT_POINT" -quiet || true
  fi
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

echo "Downloading Docker Desktop from $DOCKER_URL"
curl --fail --location --progress-bar "$DOCKER_URL" --output "$DMG_PATH"

echo "Mounting Docker.dmg"
hdiutil attach "$DMG_PATH" -nobrowse -mountpoint "$MOUNT_POINT" >/dev/null

DOCKER_APP_SOURCE="$MOUNT_POINT/Docker.app"
if [[ ! -d "$DOCKER_APP_SOURCE" ]]; then
  echo "Unable to locate Docker.app in the mounted volume." >&2
  exit 1
fi

TARGET_APP="/Applications/Docker.app"

if [[ -d "$TARGET_APP" ]]; then
  echo "Existing Docker.app found. Replacing with the latest build."
  rm -rf "$TARGET_APP"
fi

echo "Copying Docker.app to /Applications"
ditto "$DOCKER_APP_SOURCE" "$TARGET_APP"

CLI_SOURCE_DIR="$TARGET_APP/Contents/Resources/bin"
if [[ -d "$CLI_SOURCE_DIR" ]]; then
  mkdir -p /usr/local/bin
  ln -sf "$CLI_SOURCE_DIR/docker" /usr/local/bin/docker
  if [[ -f "$CLI_SOURCE_DIR/docker-compose" ]]; then
    ln -sf "$CLI_SOURCE_DIR/docker-compose" /usr/local/bin/docker-compose
  fi
  if [[ -f "$CLI_SOURCE_DIR/com.docker.cli" ]]; then
    ln -sf "$CLI_SOURCE_DIR/com.docker.cli" /usr/local/bin/com.docker.cli
  fi
fi

echo "Docker Desktop installed. Launching Docker..."
open -g "$TARGET_APP"

echo "Waiting for Docker to initialize..."
for _ in {1..30}; do
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    echo "Docker is running."
    exit 0
  fi
  sleep 5
done

echo "Docker Desktop launched, but the daemon has not reported ready yet. It may still be starting."
exit 0

