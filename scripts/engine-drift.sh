#!/usr/bin/env bash
#
# Reports engine drift between this checkout and another checkout of the
# engine (MUSGoS <-> FsosServer). Both trees are copied to a scratch directory,
# the other repo's Go module path is rewritten to this repo's, both copies are
# gofmt'd (the rewrite can reorder imports), and the result is diffed.
#
#   scripts/engine-drift.sh <other-repo> [--full]
#
# Without --full only the file list is printed. Exit status is 0 when the
# engine trees match, 1 when they drift.
set -euo pipefail

usage() { echo "Usage: $(basename "$0") <other-repo> [--full]" >&2; exit 2; }
[ $# -ge 1 ] || usage
OTHER="$(cd "$1" && pwd)"
FULL="${2:-}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# The engine: everything except the application under external/ and the
# application's own tests (FsosServer keeps them in _tests/game).
ENGINE_PATHS=(internal cmd/gameserver _tests/adapters _tests/config _tests/domain _tests/factory _tests/testutil)

module() { awk '$1 == "module" { print $2; exit }' "$1/go.mod"; }
HERE_MOD="$(module "$ROOT")"
OTHER_MOD="$(module "$OTHER")"

SCRATCH="$(mktemp -d)"
trap 'rm -rf "$SCRATCH"' EXIT

snapshot() { # <repo> <dest>
  mkdir -p "$2"
  for p in "${ENGINE_PATHS[@]}"; do
    if [ -e "$1/$p" ]; then
      mkdir -p "$2/$(dirname "$p")"
      cp -R "$1/$p" "$2/$p"
    fi
  done
}

snapshot "$ROOT" "$SCRATCH/here"
snapshot "$OTHER" "$SCRATCH/other"

if [ "$HERE_MOD" != "$OTHER_MOD" ]; then
  find "$SCRATCH/other" -name '*.go' -print0 |
    xargs -0 sed -i.bak "s#\"$OTHER_MOD/#\"$HERE_MOD/#g"
  find "$SCRATCH/other" -name '*.bak' -delete
fi
# Format each Go file explicitly so the step does not rely on directory walking.
find "$SCRATCH/here" "$SCRATCH/other" -name '*.go' -print0 | xargs -0 gofmt -w

cd "$SCRATCH"
if [ "$FULL" = "--full" ]; then
  diff -ru here other
else
  diff -rq here other
fi
