package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudriftorg/netglance/internal/notify"
	"github.com/cloudriftorg/netglance/internal/ouidb"
	"github.com/cloudriftorg/netglance/internal/store"
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

	// Trim host_events to match the cap the UI renders (HostDetail
	// slices to 100). Anything beyond is invisible storage that grows
	// forever on flapping hosts. Cheap: one indexed DELETE per scan,
	// no-op when nobody's over the cap.
	if pruned, err := st.PruneEventsPerHost(100); err != nil {
		logger.Warn("prune events", "err", err)
	} else if pruned > 0 {
		logger.Debug("pruned old events", "rows", pruned)
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

// Target is one interface+CIDR the scanner will probe, with the VLAN and name
// it will tag hosts with. Exported so the UI can show what is actually being
// scanned instead of asking the user to describe their own network back to it.
type Target struct {
	Iface  string `json:"iface"`
	CIDR   string `json:"cidr"`
	VLANID *int   `json:"vlanId,omitempty"`
	Name   string `json:"name,omitempty"`
}

// Targets resolves the current settings into the list of things that will be
// probed on the next scan. Same code path the scanner itself uses, so what the
// UI shows can't drift from what actually happens.
func Targets(s Settings) []Target {
	logger := slog.Default().With("comp", "scanner")
	found := buildScanTargets(s, ifaceAllowSet(s.ScanIfaces), logger)
	out := make([]Target, 0, len(found))
	for _, t := range found {
		name := t.name
		if name == t.iface {
			// buildScanTargets falls back to the interface name as a label;
			// that's not a user-chosen name, so report it as unnamed.
			name = ""
		}
		out = append(out, Target{Iface: t.iface, CIDR: t.cidr, VLANID: t.vlan, Name: name})
	}
	return sortTargets(out)
}

// sortTargets puts the list in VLAN order — untagged networks last, ties broken
// by CIDR. buildScanTargets walks the interface allow-list, which is a map, so
// without this the order is random per call and the UI list reshuffles.
func sortTargets(in []Target) []Target {
	sort.Slice(in, func(i, j int) bool {
		a, b := in[i], in[j]
		switch {
		case a.VLANID != nil && b.VLANID != nil && *a.VLANID != *b.VLANID:
			return *a.VLANID < *b.VLANID
		case (a.VLANID == nil) != (b.VLANID == nil):
			return a.VLANID != nil
		}
		return a.CIDR < b.CIDR
	})
	return in
}

// ifaceVLANs maps interface name → 802.1Q tag, from the environment.
// ponytail: process-global and set once in main() before the scan loop starts,
// so it needs no lock; thread it through Settings if it ever becomes editable
// at runtime.
var ifaceVLANs map[string]int

// SetIfaceVLANs installs the interface → VLAN tag map for this process.
func SetIfaceVLANs(m map[string]int) { ifaceVLANs = m }

// vlanIfaceName matches the conventional device names that carry their own tag
// — igc1_vlan20, eth0.20, vlan0.20. A bare "vlanNN" is deliberately excluded:
// on OPNsense `vlan02` is the second VLAN device, not tag 2, so reading it as a
// tag would mislabel every host on that interface.
var vlanIfaceName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(?:_vlan|\.)([0-9]{1,4})$`)

// vlanForIface resolves the VLAN of the interface a reply arrived on: the
// supplied map first, then the tag encoded in the device name. nil when
// neither knows, leaving the networks table as the fallback.
func vlanForIface(name string) *int {
	if v, ok := ifaceVLANs[name]; ok {
		return &v
	}
	if m := vlanIfaceName.FindStringSubmatch(name); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n >= 1 && n <= 4094 {
			return &n
		}
	}
	return nil
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
		// The interface wins over the network's configured VLAN: it's the
		// physical truth, the table is a guess keyed on address ranges.
		vlan := vlanForIface(iface)
		if vlan == nil {
			vlan = n.VLANID
		}
		out = append(out, scanTarget{iface: iface, cidr: n.CIDR, name: n.Name, vlan: vlan})
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
			vlan := vlanForIface(name)
			for _, n := range s.Networks {
				if n.CIDR == c {
					label = n.Name
					if vlan == nil {
						vlan = n.VLANID
					}
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
				"Netglance just discovered a device it has never seen before.\n\nIP:     %s\nMAC:    %s\nVLAN:   %s\nVendor: %s\n",
				h.IP, h.MAC, vlanLabel(h), vendorLabel(h),
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
				"A watched host stopped answering ARP scans.\n\nIP:     %s\nMAC:    %s\nVLAN:   %s\nVendor: %s\n",
				h.IP, h.MAC, vlanLabel(h), vendorLabel(h),
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
				"A watched host has reappeared on the network.\n\nIP:     %s\nMAC:    %s\nVLAN:   %s\nVendor: %s\n",
				h.IP, h.MAC, vlanLabel(h), vendorLabel(h),
			)
			if err := notify.Send(smtp, subject, body); err != nil {
				logger.Warn("notify back-online", "mac", h.MAC, "err", err)
			}
		}
	}
}

// vlanLabel mirrors the badge shown on the Hosts page: prefer the
// configured network name, fall back to "VLAN <id>", and "—" if the
// host has no VLAN tag at all.
func vlanLabel(h *store.Host) string {
	if h.NetworkName != "" {
		return h.NetworkName
	}
	if h.VLANID != nil {
		// Bare number when no name is configured — matches what the
		// host list shows in the badge.
		return fmt.Sprintf("%d", *h.VLANID)
	}
	return "—"
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
