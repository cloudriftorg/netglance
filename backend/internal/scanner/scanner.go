package scanner

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/netglance/netglance/internal/ouidb"
	"github.com/netglance/netglance/internal/store"
)

type Network struct {
	Name   string
	CIDR   string
	VLANID *int
}

type Settings struct {
	Networks         []Network
	ScanIfaces       []string // optional allow-list of interface names; empty = scan all
	ScanEnabled      bool
	ScanEverySeconds int
	OfflineAfter     int
}

type Discovery struct {
	IP          net.IP
	MAC         string
	Hostname    string
	Vendor      string
	NetworkName string
	VLANID      *int
}

type SettingsProvider func() Settings

func Run(ctx context.Context, st *store.Store, getSettings SettingsProvider) {
	logger := slog.Default().With("comp", "scanner")
	for {
		s := getSettings()
		interval := time.Duration(s.ScanEverySeconds) * time.Second
		// Floor at 10s. Anything lower turns into a broadcast-ARP storm and
		// dramatically increases false-offline flapping at OfflineAfter=1.
		if interval < 10*time.Second {
			interval = 2 * time.Minute
		}
		if s.ScanEnabled {
			if TryAcquire() {
				runOnce(ctx, st, s, logger)
				Release()
			} else {
				logger.Warn("previous scan still in flight, skipping auto cycle")
			}
		} else {
			logger.Debug("auto scan disabled, skipping cycle")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func RunOnce(ctx context.Context, st *store.Store, s Settings) (int, error) {
	return runOnce(ctx, st, s, slog.Default().With("comp", "scanner")), nil
}

func runOnce(ctx context.Context, st *store.Store, s Settings, logger *slog.Logger) int {
	startedAt := time.Now().Unix()
	all := discover(ctx, s, logger)

	for _, d := range all {
		mac := strings.ToLower(d.MAC)
		vendor := d.Vendor
		if vendor == "" || vendor == "(Unknown)" {
			vendor = ouidb.Lookup(mac)
		}
		if _, _, err := st.UpsertSeen(mac, d.IP.String(), d.NetworkName, d.VLANID, vendor, d.Hostname, startedAt); err != nil {
			logger.Warn("upsert", "mac", mac, "err", err)
		}
	}

	threshold := s.OfflineAfter
	if threshold <= 0 {
		threshold = 1
	}
	if _, err := st.MarkSweep(startedAt, threshold); err != nil {
		logger.Warn("mark sweep", "err", err)
	}

	if err := st.RecordScan(store.LastScan{
		StartedAt:  startedAt,
		EndedAt:    time.Now().Unix(),
		HostsFound: len(all),
	}); err != nil {
		logger.Warn("record scan", "err", err)
	}
	logger.Info("scan complete", "found", len(all))
	return len(all)
}

// discover runs arp-scan on the host interface that owns each configured
// network's CIDR. Same methodology as WatchYourLAN: ARP requests stay in the
// local broadcast domain, returning real MAC addresses only for hosts that
// reply. Hosts on routed VLANs require a sub-interface in that VLAN on the
// container's host (network_mode: host) — without L2 presence the network is
// silently skipped with a warning.
//
// The scan is driven by Settings.ScanIfaces — an explicit allow-list set in
// the UI. Empty list = no scan happens (the user must opt in to which NICs
// netglance touches). For each allowed iface we scan the iface's own IPv4
// CIDRs, plus any configured Network whose CIDR resolves to that iface (for
// naming and VLAN tagging).
func discover(ctx context.Context, s Settings, logger *slog.Logger) []Discovery {
	allow := ifaceAllowSet(s.ScanIfaces)
	if len(allow) == 0 {
		return nil
	}
	targets := buildScanTargets(s, allow, logger)
	if len(targets) == 0 {
		return nil
	}
	var (
		mu  sync.Mutex
		all []Discovery
		wg  sync.WaitGroup
	)
	for _, t := range targets {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			found := runArpScan(ctx, t.iface, t.cidr, logger)
			for i := range found {
				found[i].NetworkName = t.name
				found[i].VLANID = t.vlan
			}
			mu.Lock()
			all = append(all, found...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return all
}

type scanTarget struct {
	iface string
	cidr  string
	name  string
	vlan  *int
}

func ifaceAllowSet(list []string) map[string]bool {
	if len(list) == 0 {
		return nil
	}
	m := make(map[string]bool, len(list))
	for _, n := range list {
		if n != "" {
			m[n] = true
		}
	}
	return m
}

func buildScanTargets(s Settings, allow map[string]bool, logger *slog.Logger) []scanTarget {
	var out []scanTarget
	used := map[string]bool{} // iface|cidr keys to avoid duplicates

	for _, n := range s.Networks {
		iface := findInterfaceForCIDR(n.CIDR)
		if iface == "" {
			logger.Warn("no host interface in CIDR — add a VLAN sub-interface to scan it",
				"network", n.Name, "cidr", n.CIDR)
			continue
		}
		if !allow[iface] {
			logger.Debug("skipping network: iface not in scan allow-list",
				"network", n.Name, "cidr", n.CIDR, "iface", iface)
			continue
		}
		key := iface + "|" + n.CIDR
		if used[key] {
			continue
		}
		used[key] = true
		out = append(out, scanTarget{iface: iface, cidr: n.CIDR, name: n.Name, vlan: n.VLANID})
	}

	// For every allowed iface that wasn't covered by a configured Network,
	// scan its primary IPv4 network. Tag using a matching Network (by CIDR)
	// when one exists, else use the iface name as the label.
	for name := range allow {
		ifaceCidrs := ifaceIPv4Networks(name)
		for _, c := range ifaceCidrs {
			key := name + "|" + c
			if used[key] {
				continue
			}
			used[key] = true
			label := name
			var vlan *int
			for _, n := range s.Networks {
				if n.CIDR == c {
					label = n.Name
					vlan = n.VLANID
					break
				}
			}
			out = append(out, scanTarget{iface: name, cidr: c, name: label, vlan: vlan})
		}
	}
	return out
}
