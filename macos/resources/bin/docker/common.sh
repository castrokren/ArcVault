#!/bin/bash
set -euo pipefail

APP_SUPPORT_ROOT="$HOME/Library/Application Support/ArcVault"
BIN_DIR="$APP_SUPPORT_ROOT/bin"
DOCKER_DIR="$BIN_DIR/docker"
CONFIG_DIR="$APP_SUPPORT_ROOT/config"
DATA_DIR="$APP_SUPPORT_ROOT/data"
LOG_DIR="$APP_SUPPORT_ROOT/#backup_logs"
KEYS_DIR="$DATA_DIR/keys"
LOGS_DIR="$DATA_DIR/logs"

COMPOSE_FILE="$DOCKER_DIR/docker-compose.yml"
COMPOSE_ENV="$CONFIG_DIR/.env"
COMPOSE_PROJECT="arcvault"

mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR" "$KEYS_DIR" "$LOGS_DIR"

if [[ ! -f "$COMPOSE_FILE" ]];
then
  echo "Docker compose file missing: $COMPOSE_FILE" >&2
  exit 1
fi

export ARCV_APP_SUPPORT_ROOT="$APP_SUPPORT_ROOT"
export ARCV_COMPOSE_FILE="$COMPOSE_FILE"
export ARCV_COMPOSE_ENV="$COMPOSE_ENV"
export ARCV_LOG_DIR="$LOG_DIR"
export ARCV_DATA_DIR="$DATA_DIR"
export ARCV_KEYS_DIR="$KEYS_DIR"
export ARCV_DOCKER_LOG_DIR="$LOGS_DIR"

function ensure_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker CLI is not available. Please install Docker Desktop." >&2
    exit 2
  fi

  if ! docker info >/dev/null 2>&1; then
    echo "Docker is not running. Please start Docker Desktop." >&2
    exit 3
  fi
}

function compose() {
  ensure_docker
  DOCKER_COMPOSE=(docker compose)
  if ! docker compose version >/dev/null 2>&1; then
    if command -v docker-compose >/dev/null 2>&1; then
      DOCKER_COMPOSE=(docker-compose)
    else
      echo "Neither docker compose nor docker-compose is available." >&2
      exit 4
    fi
  fi

  local args=(
    --file "$COMPOSE_FILE"
    --project-name "$COMPOSE_PROJECT"
    --project-directory "$APP_SUPPORT_ROOT"
  )

  if [[ -f "$COMPOSE_ENV" ]]; then
    args+=(--env-file "$COMPOSE_ENV")
  fi

  "${DOCKER_COMPOSE[@]}" "${args[@]}" "$@"
}

function write_log() {
  local msg="$1"
  local log_file="$LOG_DIR/manager_$(date +%Y%m%d).log"
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $msg" >>"$log_file"
}
