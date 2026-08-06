package config

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ManagedFields enumerates the settings keys that the OPNsense plugin
// (or any external orchestrator) takes ownership of when NETGLANCE_MANAGED=1.
// Listed by the same JSON keys used in the /api/settings bundle so the
// frontend can directly mark them read-only.
var ManagedFields = []string{
	"httpPort",
	"scanIfaces",
}

type Config struct {
	Bind    string
	DataDir string

	// Managed reports whether the runtime config is being supplied by an
	// external orchestrator (the OPNsense plugin) via env vars. When true,
	// the API rejects writes to ManagedFields and the frontend renders
	// those fields as read-only.
	Managed bool

	// BootstrapOverrides carries the env-var-supplied values that should be
	// written into the persistent settings store at startup, before the
	// scanner loop reads them. Pointers distinguish "absent" from "set to
	// the zero value".
	Bootstrap BootstrapOverrides

	// IfaceVLANs maps an interface name to its 802.1Q tag. The interface a
	// host answers ARP on *is* its VLAN, which beats inferring one from the
	// IP: it holds no matter how the address space is laid out. Whoever runs
	// netglance supplies the map (the OPNsense plugin renders it from the
	// firewall's own VLAN config); empty means "fall back to the networks
	// table", which is what a plain host install does.
	IfaceVLANs map[string]int
}

type BootstrapOverrides struct {
	BindAddress      *string
	HTTPPort         *int
	ScanIfaces       *[]string
	ScanEverySeconds *int
	ScanEnabled      *bool
	Networks         *[]NetworkOverride
}

type NetworkOverride struct {
	CIDR   string `json:"cidr"`
	VLANID int    `json:"vlanId,omitempty"`
	Name   string `json:"name,omitempty"`
}

func Load() Config {
	c := Config{
		Bind:    envOr("NETGLANCE_BIND", ":8473"),
		DataDir: envOr("NETGLANCE_DATA_DIR", defaultDataDir()),
		Managed: envBool("NETGLANCE_MANAGED"),
	}

	if v := os.Getenv("NETGLANCE_BIND_ADDRESS"); v != "" {
		c.Bootstrap.BindAddress = &v
	}
	if v := os.Getenv("NETGLANCE_HTTP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
			c.Bootstrap.HTTPPort = &n
		}
	}
	if v := os.Getenv("NETGLANCE_SCAN_IFACES"); v != "" {
		ifaces := splitCSV(v)
		c.Bootstrap.ScanIfaces = &ifaces
	}
	if v := os.Getenv("NETGLANCE_SCAN_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 10 {
			c.Bootstrap.ScanEverySeconds = &n
		}
	}
	if v := os.Getenv("NETGLANCE_SCAN_ENABLED"); v != "" {
		b := parseBool(v)
		c.Bootstrap.ScanEnabled = &b
	}
	if v := os.Getenv("NETGLANCE_NETWORKS"); v != "" {
		nets := parseNetworks(v)
		c.Bootstrap.Networks = &nets
	}
	c.IfaceVLANs = ParseIfaceVLANs(os.Getenv("NETGLANCE_IFACE_VLANS"))

	// Compose Bind from BindAddress + HTTPPort if either is set via env.
	if c.Bootstrap.BindAddress != nil || c.Bootstrap.HTTPPort != nil {
		addr := ""
		if c.Bootstrap.BindAddress != nil && *c.Bootstrap.BindAddress != "0.0.0.0" {
			addr = *c.Bootstrap.BindAddress
		}
		port := 8473
		if c.Bootstrap.HTTPPort != nil {
			port = *c.Bootstrap.HTTPPort
		}
		c.Bind = addr + ":" + strconv.Itoa(port)
	}

	return c
}

// parseNetworks decodes the NETGLANCE_NETWORKS env var format:
//
//	cidr[:vlan[:name]],...
//
// Examples:
//
//	"10.0.0.0/24"
//	"10.0.0.0/24:10:lan,192.168.1.0/24:0:guest"
func parseNetworks(s string) []NetworkOverride {
	var out []NetworkOverride
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, ":", 3)
		n := NetworkOverride{CIDR: strings.TrimSpace(parts[0])}
		if n.CIDR == "" {
			continue
		}
		if len(parts) >= 2 {
			if v, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				n.VLANID = v
			}
		}
		if len(parts) >= 3 {
			n.Name = strings.TrimSpace(parts[2])
		}
		out = append(out, n)
	}
	return out
}

// ParseIfaceVLANs decodes the NETGLANCE_IFACE_VLANS env var format:
//
//	iface=tag,iface=tag
//
// Example: "vlan02=10,vlan03=20,igc1_vlan90=90". Malformed pairs and
// out-of-range tags are dropped rather than failing the whole map — a typo in
// one entry shouldn't cost you the VLAN labelling of every other interface.
func ParseIfaceVLANs(s string) map[string]int {
	out := map[string]int{}
	for _, raw := range strings.Split(s, ",") {
		name, tag, ok := strings.Cut(strings.TrimSpace(raw), "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		n, err := strconv.Atoi(strings.TrimSpace(tag))
		if name == "" || err != nil || n < 1 || n > 4094 {
			continue
		}
		out[name] = n
	}
	return out
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envBool(k string) bool {
	return parseBool(os.Getenv(k))
}

// defaultDataDir resolves the on-disk location for the SQLite database when
// NETGLANCE_DATA_DIR isn't set. FreeBSD picks the OS-conventional /var/db
// (where the netglance pkg pre-creates the dir); Linux containers keep /data
// (the historical Docker mount); everything else falls back to ./data for
// local dev.
func defaultDataDir() string {
	if runtime.GOOS == "freebsd" {
		if _, err := os.Stat("/var/db/netglance"); err == nil {
			return "/var/db/netglance"
		}
	}
	if _, err := os.Stat("/data"); err == nil {
		return "/data"
	}
	return "./data"
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
