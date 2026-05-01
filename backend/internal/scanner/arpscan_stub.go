//go:build !linux && !freebsd

package scanner

import (
	"context"
	"log/slog"
)

// Stubs so the package builds on platforms without arp-scan support
// (e.g. macOS dev machines / CI). Supported runtime targets are Linux
// (Docker / native) and FreeBSD (OPNsense plugin / native), both of
// which provide the unix implementation in arpscan_unix.go.

func runArpScan(_ context.Context, _ string, _ string, _ *slog.Logger) []Discovery {
	return nil
}

func findInterfaceForCIDR(_ string) string { return "" }

func ifaceIPv4Networks(_ string) []string { return nil }
