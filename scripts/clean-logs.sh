#!/bin/bash
# Delete every *.log file under $HOME (hidden directories included), BATCH
# files per rm call. Deleted files are not recoverable; use --dry-run first.
set -uo pipefail

BATCH="${BATCH:-100}"
dry_run=false
case "${1:-}" in
  "") ;;
  --dry-run) dry_run=true ;;
  *)
    echo "Usage: [BATCH=100] bash scripts/clean-logs.sh [--dry-run]" >&2
    exit 1
    ;;
esac
if ! [[ "$BATCH" =~ ^[1-9][0-9]*$ ]]; then
  echo "BATCH must be a positive integer, got \"$BATCH\"" >&2
  exit 1
fi

# find exits non-zero on unreadable directories (macOS-protected ones under
# ~/Library); those are skipped, not fatal.
if $dry_run; then
  find "$HOME" -type f -name '*.log' 2>/dev/null
  exit 0
fi
find "$HOME" -type f -name '*.log' -print0 2>/dev/null |
  xargs -0 -n "$BATCH" rm -f --
