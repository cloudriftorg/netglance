//go:build linux || freebsd

package scanner

import (
	"bufio"
	"bytes"
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
const arpScanTimeout = 120 * time.Second

// runArpScan executes `arp-scan -gNx -r 5 -I <iface> <cidr>` and returns one
// Discovery per responding host. We pass the explicit CIDR (instead of the
// `-l` localnet flag) so the scan scope matches exactly what the user
// configured in netglance, regardless of the interface's own netmask.
//
// `-r 5` (retry) is bumped from the arp-scan default of 3: hosts on a
// quiet/slow VLAN often miss the first couple of ARP requests when the
// container has just started, and the difference between 3 and 5 retries
// is negligible on a healthy LAN but recovers a lot of devices on a
// loaded one.
func runArpScan(parent context.Context, iface, cidr string, logger *slog.Logger) []Discovery {
	ctx, cancel := context.WithTimeout(parent, arpScanTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "arp-scan", "-gNx", "-r", "5", "-I", iface, cidr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logger.Warn("arp-scan failed",
			"iface", iface, "cidr", cidr, "err", err,
			"stderr", strings.TrimSpace(stderr.String()))
		return nil
	}
	if s := strings.TrimSpace(stderr.String()); s != "" {
		logger.Debug("arp-scan stderr", "iface", iface, "cidr", cidr, "msg", s)
	}
	hosts := parseArpScanOutput(stdout.String())
	logger.Debug("arp-scan ok", "iface", iface, "cidr", cidr, "found", len(hosts))
	return hosts
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

// maxAutoPrefix is the smallest prefix length (largest subnet) we'll derive
// from an interface's own netmask. Anything wider (e.g. Docker's /16) is
// narrowed to a /24 centred on the interface's IP — arp-scanning 65k hosts
// busts the per-invocation timeout and produces nothing useful.
const maxAutoPrefix = 22

// ifaceIPv4Networks returns the CIDR of every IPv4 address bound to the named
// interface (e.g. "192.168.1.0/24"). Used when the user picks an iface to scan
// without configuring a matching CIDR — we derive the scope from the iface
// itself.
func ifaceIPv4Networks(name string) []string {
	ifc, err := net.InterfaceByName(name)
	if err != nil {
		return nil
	}
	if ifc.Flags&net.FlagUp == 0 {
		return nil
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, a := range addrs {
		ipn, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipn.IP.To4()
		if ip4 == nil || ip4.IsLinkLocalUnicast() {
			continue
		}
		mask := ipn.Mask
		if ones, bits := mask.Size(); bits == 32 && ones < maxAutoPrefix {
			mask = net.CIDRMask(24, 32)
		}
		network := &net.IPNet{IP: ip4.Mask(mask), Mask: mask}
		s := network.String()
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
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
