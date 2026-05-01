//go:build !linux

package scanner

import (
	"context"
	"log/slog"
)

// Stubs so the package builds on non-Linux platforms (CI on macOS runners).
// The supported runtime target is a Linux container with arp-scan installed.

func runArpScan(_ context.Context, _ string, _ string, _ *slog.Logger) []Discovery {
	return nil
}

func findInterfaceForCIDR(_ string) string { return "" }

func ifaceIPv4Networks(_ string) []string { return nil }

func activeIPv4Ifaces() []string { return nil }
