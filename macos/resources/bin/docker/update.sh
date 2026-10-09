#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

write_log "Updating ArcVault stack"
compose pull
compose up -d --build
write_log "ArcVault stack updated"
echo "ArcVault services updated"
