#!/usr/bin/env bash
set -euo pipefail

pkill -f 'bin/hangar-agent' 2>/dev/null || true
pkill -f 'hangar-agent -config' 2>/dev/null || true
if command -v fuser >/dev/null 2>&1; then
  fuser -k 8741/tcp 2>/dev/null || true
fi
echo "agent stopped (if it was running)"
