#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/common.sh"

ensure_docker || exit $?

SERVICES=$(compose ps --services 2>/dev/null || true)

if [[ -z "$SERVICES" ]]; then
  python3 - <<'PY'
import json
print(json.dumps({
    "health": "Not Running",
    "details": "ArcVault services are not currently running"
}))
PY
  exit 0
fi

SERVICES_JSON=$(compose ps --format json 2>/dev/null || echo "")

if [[ -z "$SERVICES_JSON" ]]; then
  STATUS_TEXT=$(compose ps 2>/dev/null || echo "ArcVault services appear to be running")
  export STATUS_TEXT
  python3 - <<'PY'
import json, os
details = os.environ.get("STATUS_TEXT", "")
print(json.dumps({
    "health": "Running",
    "details": details
}))
PY
  exit 0
fi

export SERVICES_JSON
python3 - <<'PY'
import json
import os

raw = os.environ["SERVICES_JSON"]
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    print(json.dumps({"health": "Unknown", "details": raw}))
    raise SystemExit(0)

if not data:
    print(json.dumps({
        "health": "Unknown",
        "details": "No services reported by docker compose"
    }))
    raise SystemExit(0)

details = []
overall = "Running"
for service in data:
    health = (service.get("Health") or "").lower()
    state = (service.get("State") or "").lower()
    label = service.get("Service")

    details.append({
        "service": label,
        "state": state,
        "health": health,
    })

    if "unhealthy" in health or "exited" in state:
        overall = "Unhealthy"
    elif "starting" in health or "created" in state:
        if overall != "Unhealthy":
            overall = "Starting"

print(json.dumps({
    "health": overall,
    "details": details
}))
PY
