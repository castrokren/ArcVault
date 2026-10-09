#!/usr/bin/env bash
set -euo pipefail

# Build a self-contained Linux binary for the controller using PyInstaller.
# Can be run from any directory; resolves repo root from this script's path.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"
cd "$REPO_ROOT"

python3 -m venv .venv_controller_linux
source .venv_controller_linux/bin/activate

pip install --upgrade pip
pip install pyinstaller
pip install -e apps/backup-control

pyinstaller --noconfirm --clean --onefile \
  --name robobackup-controller \
  apps/backup-control/controller/run_controller.py \
  --add-data "apps/backup-control/controller:apps/backup-control/controller"

echo
echo "Build complete. Output: dist/robobackup-controller"


