#!/usr/bin/env bash
#
# Test runner with three modes:
#   --unit          only unit tests (fast, no Docker)
#   --integration   only integration tests (brings up third-party services)
#   --all (default) unit + integration
#
# Integration tests live behind the `integration` build tag and have
# "Integration" in their name; the --integration mode filters on that so unit
# tests aren't re-run.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TP="$ROOT/docker/thirdparties"
ENV_FILE="$TP/custom_settings.env"

TEST_PKGS=(./_tests/config/... ./_tests/domain/... ./_tests/factory/... ./_tests/adapters/...)
INTEGRATION_RUN='Integration'

mode="all"
case "${1:-}" in
  --unit)         mode="unit" ;;
  --integration)  mode="integration" ;;
  --all|"")       mode="all" ;;
  -h|--help)      echo "Usage: $(basename "$0") [--unit|--integration|--all]"; exit 0 ;;
  *) echo "error: unknown argument '$1'" >&2; exit 1 ;;
esac

cd "$ROOT"

if [ "$mode" = "unit" ]; then
  exec go test "${TEST_PKGS[@]}"
fi

# integration / all: make sure the third-party services are running.
"$TP/run-thirdparties-docker.sh"

# Derive the TEST_* connection env the Go suites read from custom_settings.env
# (already-set env wins, so CI can override without editing the file).
set -a
# shellcheck disable=SC1090
[ -f "$ENV_FILE" ] && . "$ENV_FILE"
set +a

export TEST_POSTGRES_DSN="${TEST_POSTGRES_DSN:-postgres://${MUS_PG_USER:-postgres}:${MUS_PG_PASSWORD:-my_secret_pw}@127.0.0.1:${MUS_PG_PORT:-5432}/${MUS_PG_DB:-musgo_regression}?sslmode=disable}"
export TEST_REDIS_ADDR="${TEST_REDIS_ADDR:-127.0.0.1:${MUS_REDIS_PORT:-6379}}"
export TEST_RABBITMQ_HOST="${TEST_RABBITMQ_HOST:-127.0.0.1}"
export TEST_RABBITMQ_PORT="${TEST_RABBITMQ_PORT:-${MUS_RABBITMQ_PORT:-5672}}"
export TEST_RABBITMQ_USER="${TEST_RABBITMQ_USER:-${MUS_RABBITMQ_USER:-guest}}"
export TEST_RABBITMQ_PASSWORD="${TEST_RABBITMQ_PASSWORD:-${MUS_RABBITMQ_PASSWORD:-guest}}"
export TEST_RABBITMQ_VHOST="${TEST_RABBITMQ_VHOST:-/}"

if [ "$mode" = "integration" ]; then
  exec go test -tags=integration -run "$INTEGRATION_RUN" "${TEST_PKGS[@]}"
fi

# all
exec go test -tags=integration "${TEST_PKGS[@]}"
