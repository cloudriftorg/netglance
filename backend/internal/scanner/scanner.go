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
	ScanEverySeconds int
	OfflineAfter     int
	PrimaryIface     string
}

type Discovery struct {
	IP          net.IP
	MAC         string
	Hostname    string
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
			interval = 5 * time.Minute
		}
		runOnce(ctx, st, s, logger)
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
	scanID, err := st.StartScan("")
	if err != nil {
		logger.Error("start scan", "err", err)
		return 0
	}
	now := time.Now().Unix()
	all := discover(ctx, s, logger)

	for _, d := range all {
		mac := strings.ToLower(d.MAC)
		vendor := ouidb.Lookup(mac)
		if _, _, err := st.UpsertSeen(mac, d.IP.String(), d.NetworkName, d.VLANID, vendor, d.Hostname, now); err != nil {
			logger.Warn("upsert", "mac", mac, "err", err)
		}
	}

	threshold := s.OfflineAfter
	if threshold <= 0 {
		threshold = 3
	}
	if _, err := st.MarkSweep(now, threshold); err != nil {
		logger.Warn("mark sweep", "err", err)
	}

	if err := st.FinishScan(scanID, len(all), ""); err != nil {
		logger.Warn("finish scan", "err", err)
	}
	logger.Info("scan complete", "found", len(all))
	return len(all)
}

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
			found := scanNetwork(ctx, n, logger)
			mu.Lock()
			all = append(all, found...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if arpEntries := readARPTable(); len(arpEntries) > 0 {
		for i, d := range all {
			if d.MAC == "" {
				if mac, ok := arpEntries[d.IP.String()]; ok {
					all[i].MAC = mac
				}
			}
		}
	}
	out := all[:0]
	for _, d := range all {
		if d.MAC != "" {
			out = append(out, d)
		}
	}
	return out
}
