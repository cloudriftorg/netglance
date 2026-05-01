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

// listInterfacesHandler returns active, non-loopback interfaces with at least
// one IPv4 address. Used by the Settings UI to let users pick an explicit
// interface for a given network instead of relying on CIDR auto-detection.
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
