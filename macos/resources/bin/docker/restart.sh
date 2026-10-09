#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

write_log "Restarting ArcVault stack"
compose down
compose up -d
write_log "ArcVault stack restarted"
echo "ArcVault services restarted"
