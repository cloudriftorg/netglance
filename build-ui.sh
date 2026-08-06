#!/bin/sh
#
# Build the React UI and stage it where //go:embed picks it up. Shared by
# build.sh and build-native.sh so the two can't disagree about where the
# frontend lives.

set -eu

cd "$(dirname "$0")"

echo "==> building frontend"
(cd frontend && npm install --no-audit --no-fund && npm run build)
rm -rf backend/internal/webui/dist
mkdir -p backend/internal/webui/dist
cp -R frontend/dist/. backend/internal/webui/dist/
# Tracked, and the only file in here git knows about — see .gitignore.
touch backend/internal/webui/dist/.gitkeep
