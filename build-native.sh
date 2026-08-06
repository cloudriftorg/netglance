#!/bin/sh
#
# Build a tarball for a plain host — no OPNsense, no plugin. Binary plus a
# systemd unit; you install it by hand (see the README).
#
#   ./build-native.sh                  → dist/netglance-linux-amd64.tar.gz
#   ./build-native.sh linux arm64
#   ./build-native.sh freebsd amd64
#
# arp-scan is NOT bundled here: unlike OPNsense, a normal distro has it a
# package manager away (`apt install arp-scan`, `pkg install arp-scan`).

set -eu

cd "$(dirname "$0")"

GOOS="${1:-linux}"
GOARCH="${2:-amd64}"
case "$GOOS" in
    linux|freebsd) ;;
    *) echo "unsupported OS: $GOOS (use linux or freebsd)" >&2; exit 1 ;;
esac
case "$GOARCH" in
    amd64|arm64) ;;
    *) echo "unsupported arch: $GOARCH (use amd64 or arm64)" >&2; exit 1 ;;
esac

for tool in go npm tar; do
    command -v "$tool" >/dev/null 2>&1 || { echo "missing required tool: $tool" >&2; exit 1; }
done

./build-ui.sh

echo "==> cross-compiling netglance for $GOOS/$GOARCH"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
NAME="netglance-$GOOS-$GOARCH"
STAGE=$(mktemp -d "${TMPDIR:-/tmp}/netglance-native.XXXXXX")
trap 'rm -rf "$STAGE"' EXIT INT TERM
mkdir -p "$STAGE/$NAME"
(cd backend && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
    -o "$STAGE/$NAME/netglance" ./cmd/server)
cp deploy/native/netglance.service "$STAGE/$NAME/"

mkdir -p dist
OUT="dist/$NAME.tar.gz"
tar --no-mac-metadata --no-xattrs -czf "$OUT" -C "$STAGE" "$NAME"

echo "✓ $OUT  ($(du -h "$OUT" | cut -f1), netglance $VERSION)"
echo
echo "  copy it to the host and follow 'Native install' in the README"
