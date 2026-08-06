#!/bin/sh
#
# Build the self-contained OPNsense installer.
#
#   ./build.sh            → dist/installer.sh (amd64, pkg ABI FreeBSD:15)
#   ./build.sh arm64      → same, for an ARM OPNsense box
#   ./build.sh amd64 14   → for OPNsense 25.x, whose pkg ABI is FreeBSD:14
#
# Needs here: go, node/npm. Needs nothing on the OPNsense box — copy the
# resulting file over however you like and run it there:
#
#   sh installer.sh              install or update
#   sh installer.sh uninstall    remove every trace

set -eu

cd "$(dirname "$0")"

GOARCH="${1:-amd64}"
case "$GOARCH" in
    amd64|arm64) ;;
    *) echo "unsupported arch: $GOARCH (use amd64 or arm64)" >&2; exit 1 ;;
esac

# The bundled arp-scan package must match the target's pkg ABI, and pkg refuses
# a mismatch outright. Check it on the box with `pkg config abi` — OPNsense 26.x
# is FreeBSD:15, 25.x was FreeBSD:14.
FBSD="${2:-15}"

for tool in go npm tar; do
    command -v "$tool" >/dev/null 2>&1 || { echo "missing required tool: $tool" >&2; exit 1; }
done

# The Go binary embeds the built UI (//go:embed), so the firewall gets one
# file instead of a binary plus an asset tree to keep in sync.
./build-ui.sh

echo "==> cross-compiling netglance for freebsd/$GOARCH"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
STAGE=$(mktemp -d "${TMPDIR:-/tmp}/netglance-build.XXXXXX")
trap 'rm -rf "$STAGE"' EXIT INT TERM
# CGO_ENABLED=0 keeps it a static binary (SQLite here is pure Go), so it has
# no library expectations on the target at all.
(cd backend && CGO_ENABLED=0 GOOS=freebsd GOARCH="$GOARCH" \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
    -o "$STAGE/netglance" ./cmd/server)

# arp-scan does the actual L2 sweep, and OPNsense's curated repo doesn't carry
# it (930 packages, arp-scan isn't one) — so we ship FreeBSD's package inside
# the installer and `pkg add` it offline. It needs no other package: its only
# shared libraries, libc and libpcap, are in the FreeBSD base OPNsense runs on.
# The version is never pinned: the repo index says which one is current, and
# the cached package is named after it, so a new upstream release simply isn't
# in the cache and gets fetched. The index itself is refreshed weekly — that's
# the only staleness, and `rm -rf dist/deps` clears it.
case "$GOARCH" in
    amd64) ABI="FreeBSD:$FBSD:amd64" ;;
    arm64) ABI="FreeBSD:$FBSD:aarch64" ;;
esac
REPO="https://pkg.freebsd.org/$ABI/latest"
INDEX="dist/deps/packagesite-$ABI.pkg"
mkdir -p dist/deps
if [ -z "$(find "$INDEX" -mtime -7 2>/dev/null)" ]; then
    echo "==> refreshing the FreeBSD $ABI package index"
    curl -fsSL -o "$INDEX" "$REPO/packagesite.pkg"
fi
REPOPATH=$(tar -xOf "$INDEX" packagesite.yaml \
    | grep -o '{"name":"arp-scan".*' \
    | sed -n 's/.*"repopath":"\([^"]*\)".*/\1/p' | head -1)
[ -n "$REPOPATH" ] || { echo "arp-scan not found in the FreeBSD $ABI repo" >&2; exit 1; }
ARP_PKG="dist/deps/$(basename "$REPOPATH")"
if [ ! -f "$ARP_PKG" ]; then
    echo "==> fetching $(basename "$REPOPATH") for $ABI"
    curl -fsSL -o "$ARP_PKG" "$REPO/$REPOPATH"
fi
cp "$ARP_PKG" "$STAGE/arp-scan.pkg"

echo "==> assembling installer"
cp -R deploy/opnsense-plugin/src "$STAGE/src"
find "$STAGE/src" \( -name '.DS_Store' -o -name '._*' \) -delete

mkdir -p dist
OUT=dist/installer.sh
cat deploy/opnsense-plugin/installer.sh > "$OUT"
echo "__PAYLOAD_BELOW__" >> "$OUT"
# --no-mac-metadata --no-xattrs: macOS tar would otherwise record
# com.apple.provenance on every entry, and FreeBSD's tar fails to restore it —
# which under `set -e` killed the installer right after unpacking.
# Byte-identical rebuilds when nothing changed: tar records each entry's mtime
# and ownership, and the staged files are freshly created on every run. Without
# this, every build shows up as a git diff on the committed installer.
find "$STAGE" -exec touch -t 200001010000 {} +
# gzip -n drops the timestamp it would otherwise stamp into the stream.
tar --no-mac-metadata --no-xattrs --uid 0 --gid 0 --uname root --gname root -cf - -C "$STAGE" netglance arp-scan.pkg src \
    | gzip -n -9 >> "$OUT"
chmod +x "$OUT"

echo "✓ $OUT  ($(du -h "$OUT" | cut -f1), netglance $VERSION, freebsd/$GOARCH)"
echo
echo "  copy it to the firewall, then run it there as root:"
echo "    sh installer.sh              install or update"
echo "    sh installer.sh uninstall    remove every trace"
