#!/bin/sh
# Helper: push the local plugin tree to an OPNsense VM and restart configd.
# Usage: VM_HOST=root@10.0.0.42 ./dev/opnsense-vm/sync.sh
#
# Equivalent to `make dev-plugin-sync`; standalone for cases where you don't
# want to invoke make (e.g. from an editor's "on save" hook).

set -eu

if [ -z "${VM_HOST:-}" ]; then
    echo "ERROR: set VM_HOST=root@<ip-of-opnsense-vm>" >&2
    exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SRC="$REPO_ROOT/deploy/opnsense-plugin/src/"

rsync -av --delete \
    --exclude='.DS_Store' --exclude='._*' \
    "$SRC" "$VM_HOST:/usr/local/opnsense/"

ssh "$VM_HOST" 'service configd restart && (configctl netglance reconfigure 2>/dev/null || true)'
echo "✓ synced and reloaded on $VM_HOST"
