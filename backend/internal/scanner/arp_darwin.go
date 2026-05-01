//go:build darwin

package scanner

import (
	"net"
	"os/exec"
	"regexp"
	"strings"
)

var arpLine = regexp.MustCompile(`\(([\d.]+)\) at ([0-9a-fA-F:]+)`)

func readARPTable() map[string]string {
	cmd := exec.Command("/usr/sbin/arp", "-an")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		m := arpLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		mac := strings.ToLower(m[2])
		if mac == "" || mac == "(incomplete)" || strings.Count(mac, ":") != 5 {
			continue
		}
		parts := strings.Split(mac, ":")
		for i, p := range parts {
			if len(p) == 1 {
				parts[i] = "0" + p
			}
		}
		result[m[1]] = strings.Join(parts, ":")
	}
	return result
}

func autoDetectCIDR() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 || ifc.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			mask := net.CIDRMask(24, 32)
			return (&net.IPNet{IP: ip4.Mask(mask), Mask: mask}).String()
		}
	}
	return ""
}
