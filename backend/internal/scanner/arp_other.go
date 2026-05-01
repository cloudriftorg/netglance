//go:build !linux && !darwin

package scanner

import "net"

func readARPTable() map[string]string { return nil }

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
