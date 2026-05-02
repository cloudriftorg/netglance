package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/netglance/netglance/internal/notify"
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
	// Notify holds the SMTP config + which transitions should fire emails.
	// nil = notifications fully disabled (e.g. SMTP not configured).
	Notify *NotifyConfig
}

type NotifyConfig struct {
	SMTP        SMTPConfig
	OnNewHost   bool
	OnOffline   bool
	OnBackOnline bool
}

type SMTPConfig struct {
	Host       string
	Port       int
	UseTLS     bool
	UseAuth    bool
	Username   string
	Password   string
	From       string
	Recipients []string
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

	var newHosts []*store.Host
	var backOnlineHosts []*store.Host
	for _, d := range all {
		mac := strings.ToLower(d.MAC)
		vendor := d.Vendor
		if vendor == "" || vendor == "(Unknown)" {
			vendor = ouidb.Lookup(mac)
		}
		h, isNew, wasOffline, err := st.UpsertSeen(mac, d.IP.String(), d.NetworkName, d.VLANID, vendor, d.Hostname, startedAt)
		if err != nil {
			logger.Warn("upsert", "mac", mac, "err", err)
			continue
		}
		if isNew {
			newHosts = append(newHosts, h)
		} else if wasOffline {
			backOnlineHosts = append(backOnlineHosts, h)
		}
	}

	threshold := s.OfflineAfter
	if threshold <= 0 {
		threshold = 1
	}
	wentOffline, err := st.MarkSweep(startedAt, threshold)
	if err != nil {
		logger.Warn("mark sweep", "err", err)
	}

	// Fire notification emails after the DB writes are durable. Each
	// transition is a separate concern: new-host applies regardless of
	// per-host watch flag, offline/back-online only fire when the host
	// has notify_offline = 1.
	notifyTransitions(s.Notify, newHosts, wentOffline, backOnlineHosts, logger)

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

// notifyTransitions emails the configured recipients about new / offline /
// back-online host transitions detected during this scan tick. Skips
// silently when SMTP isn't configured or no recipients are set, so the
// scanner stays quiet on a fresh install.
func notifyTransitions(cfg *NotifyConfig, newHosts, wentOffline, backOnline []*store.Host, logger *slog.Logger) {
	if cfg == nil || cfg.SMTP.Host == "" || len(cfg.SMTP.Recipients) == 0 {
		return
	}
	smtp := notify.Config{
		Host:       cfg.SMTP.Host,
		Port:       cfg.SMTP.Port,
		UseTLS:     cfg.SMTP.UseTLS,
		UseAuth:    cfg.SMTP.UseAuth,
		Username:   cfg.SMTP.Username,
		Password:   cfg.SMTP.Password,
		From:       cfg.SMTP.From,
		Recipients: cfg.SMTP.Recipients,
	}
	if cfg.OnNewHost {
		for _, h := range newHosts {
			subject := fmt.Sprintf("Netglance — new device on LAN: %s", hostLabel(h))
			body := fmt.Sprintf(
				"Netglance just discovered a device it has never seen before.\n\nIP:     %s\nMAC:    %s\nVendor: %s\n",
				h.IP, h.MAC, vendorLabel(h),
			)
			if err := notify.Send(smtp, subject, body); err != nil {
				logger.Warn("notify new", "mac", h.MAC, "err", err)
			}
		}
	}
	if cfg.OnOffline {
		for _, h := range wentOffline {
			if !h.NotifyOffline {
				continue // only watched hosts trigger offline mails
			}
			subject := fmt.Sprintf("Netglance — host went offline: %s", hostLabel(h))
			body := fmt.Sprintf(
				"A watched host stopped answering ARP scans.\n\nIP:     %s\nMAC:    %s\nVendor: %s\n",
				h.IP, h.MAC, vendorLabel(h),
			)
			if err := notify.Send(smtp, subject, body); err != nil {
				logger.Warn("notify offline", "mac", h.MAC, "err", err)
			}
		}
	}
	if cfg.OnBackOnline {
		for _, h := range backOnline {
			if !h.NotifyOnline {
				continue // separate per-host opt-in for back-online emails
			}
			subject := fmt.Sprintf("Netglance — host back online: %s", hostLabel(h))
			body := fmt.Sprintf(
				"A watched host has reappeared on the network.\n\nIP:     %s\nMAC:    %s\nVendor: %s\n",
				h.IP, h.MAC, vendorLabel(h),
			)
			if err := notify.Send(smtp, subject, body); err != nil {
				logger.Warn("notify back-online", "mac", h.MAC, "err", err)
			}
		}
	}
}

func hostLabel(h *store.Host) string {
	if h.CustomName != "" {
		return h.CustomName
	}
	return h.IP
}

func vendorLabel(h *store.Host) string {
	if h.CustomVendor != "" {
		return h.CustomVendor
	}
	if h.Vendor != "" {
		return h.Vendor
	}
	return "(unknown)"
}
