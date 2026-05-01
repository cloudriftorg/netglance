//go:build linux

package scanner

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"time"
)

// arpScanTimeout caps how long a single arp-scan invocation can run. Without
// this, a misconfigured interface (e.g. a Docker bridge with a /16 netmask
// in dev) makes arp-scan hang for many minutes scanning 65k IPs, which in
// turn pins the global inFlight lock and stalls the periodic scan loop.
const arpScanTimeout = 60 * time.Second

// runArpScan executes `arp-scan -gNx -I <iface> <cidr>` and returns one
// Discovery per responding host. We pass the explicit CIDR (instead of the
// `-l` localnet flag) so the scan scope matches exactly what the user
// configured in netglance, regardless of the interface's own netmask.
func runArpScan(parent context.Context, iface, cidr string, logger *slog.Logger) []Discovery {
	ctx, cancel := context.WithTimeout(parent, arpScanTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "arp-scan", "-gNx", "-I", iface, cidr)
	out, err := cmd.Output()
	if err != nil {
		logger.Warn("arp-scan failed", "iface", iface, "cidr", cidr, "err", err)
		return nil
	}
	return parseArpScanOutput(string(out))
}

func parseArpScanOutput(text string) []Discovery {
	var hosts []Discovery
	s := bufio.NewScanner(strings.NewReader(text))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		ip := net.ParseIP(strings.TrimSpace(parts[0]))
		mac := strings.ToLower(strings.TrimSpace(parts[1]))
		if ip == nil || mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}
		d := Discovery{IP: ip, MAC: mac}
		if len(parts) >= 3 {
			d.Vendor = strings.TrimSpace(parts[2])
		}
		hosts = append(hosts, d)
	}
	return hosts
}

// findInterfaceForCIDR returns the host's interface name that owns an IPv4
// address inside the given CIDR. arp-scan needs an interface with L2
// presence in the target subnet — we resolve it from the configured CIDR
// so the user keeps configuring networks by CIDR (as before) instead of
// by raw iface name.
func findInterfaceForCIDR(cidr string) string {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ""
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil {
				continue
			}
			if ipnet.Contains(ip4) {
				return ifc.Name
			}
		}
	}
	return ""
}
