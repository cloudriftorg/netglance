package api

import (
	"net"
	"net/http"
	"sort"
	"strings"
)

type ifaceInfo struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
}

// virtualBridgePrefixes lists name prefixes / exact names that we hide from
// the interface picker. These are Docker-managed bridges, container vNICs,
// and similar virtual interfaces — scanning a /16 bridge with arp-scan just
// times out and produces nothing useful.
var virtualBridgePrefixes = []string{"br-", "veth", "virbr", "cni", "flannel", "cali", "tailscale", "wg", "tun", "tap"}

func isVirtualIface(name string) bool {
	if name == "docker0" {
		return true
	}
	for _, p := range virtualBridgePrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// listInterfacesHandler returns active, non-loopback interfaces with at least
// one IPv4 address. Used by Settings (and the second step of the setup wizard)
// to let users pick which interfaces netglance should scan. Authentication is
// enforced via the parent route group — this is the only sensitive action
// here, and the setup wizard hits it after the admin password is set.
func listInterfacesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, listIfaces())
	}
}

func listIfaces() []ifaceInfo {
	out := []ifaceInfo{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualIface(ifc.Name) {
			continue
		}
		addrs, _ := ifc.Addrs()
		var v4 []string
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() {
				continue
			}
			v4 = append(v4, ipn.String())
		}
		if len(v4) == 0 {
			continue
		}
		out = append(out, ifaceInfo{Name: ifc.Name, Addresses: v4})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
