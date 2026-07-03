#!/usr/bin/env bash
#
# Bring up (default) or tear down the third-party services that the integration
# test suites run against: Postgres, Redis, RabbitMQ.
#
# Mirrors the Apache Doris docker/thirdparties approach: one compose file per
# service, selectable, driven by custom_settings.env. The Go integration suites
# connect to whatever this script starts.
#
# Usage:
#   run-thirdparties-docker.sh                    # start all
#   run-thirdparties-docker.sh -c redis,rabbitmq  # start a subset
#   run-thirdparties-docker.sh --stop             # stop + remove all (with volumes)
#   run-thirdparties-docker.sh -c postgres --stop # stop a subset
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$SCRIPT_DIR/docker-compose"
ENV_FILE="$SCRIPT_DIR/custom_settings.env"
PROJECT_PREFIX="${MUS_COMPOSE_PROJECT:-musgo-tp}"

ALL=(postgres redis rabbitmq)
components=""
action="up"

usage() { sed -n '2,15p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }

while [ $# -gt 0 ]; do
  case "$1" in
    -c|--components) components="${2:-}"; shift 2 ;;
    --stop|--down)   action="down"; shift ;;
    -h|--help)       usage; exit 0 ;;
    *) echo "error: unknown argument '$1'" >&2; usage; exit 1 ;;
  esac
done

# Pick the available compose command (v2 plugin preferred).
if docker compose version >/dev/null 2>&1; then
  DC=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  DC=(docker-compose)
else
  echo "error: neither 'docker compose' nor 'docker-compose' is installed" >&2
  exit 1
fi

if [ -n "$components" ]; then
  IFS=',' read -r -a SELECTED <<< "$components"
else
  SELECTED=("${ALL[@]}")
fi

for comp in "${SELECTED[@]}"; do
  file="$COMPOSE_DIR/$comp/$comp.yaml"
  if [ ! -f "$file" ]; then
    echo "error: no compose file for component '$comp' ($file)" >&2
    exit 1
  fi
  project="${PROJECT_PREFIX}-${comp}"
  if [ "$action" = "down" ]; then
    echo ">> stopping $comp"
    "${DC[@]}" -p "$project" -f "$file" --env-file "$ENV_FILE" down -v
  else
    echo ">> starting $comp (waiting until healthy)"
    "${DC[@]}" -p "$project" -f "$file" --env-file "$ENV_FILE" up -d --wait
  fi
done

if [ "$action" = "up" ]; then
  echo ""
  echo "third-party services are up. Connection settings: $ENV_FILE"
  echo "Run the integration suite with: make test-integration"
fi
