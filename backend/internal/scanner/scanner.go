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
		if interval < 30*time.Second {
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
func discover(ctx context.Context, s Settings, logger *slog.Logger) []Discovery {
	if len(s.Networks) == 0 {
		if cidr := autoDetectCIDR(); cidr != "" {
			s.Networks = []Network{{Name: "auto", CIDR: cidr}}
		}
	}
	var (
		mu  sync.Mutex
		all []Discovery
		wg  sync.WaitGroup
	)
	for _, n := range s.Networks {
		n := n
		wg.Add(1)
		go func() {
			defer wg.Done()
			iface := findInterfaceForCIDR(n.CIDR)
			if iface == "" {
				logger.Warn("no host interface in CIDR — add a VLAN sub-interface to scan it",
					"network", n.Name, "cidr", n.CIDR)
				return
			}
			found := runArpScan(ctx, iface, n.CIDR, logger)
			for i := range found {
				found[i].NetworkName = n.Name
				found[i].VLANID = n.VLANID
			}
			mu.Lock()
			all = append(all, found...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return all
}
