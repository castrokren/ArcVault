#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

write_log "Stopping ArcVault stack"
compose down
write_log "ArcVault stack stopped"
echo "ArcVault services have been stopped"
