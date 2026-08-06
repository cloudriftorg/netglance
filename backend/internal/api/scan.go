package api

import (
	"context"
	"net/http"
	"time"

	"github.com/cloudriftorg/netglance/internal/scanner"
	"github.com/cloudriftorg/netglance/internal/store"
)

func runScanHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !kickScan(st) {
			writeJSON(w, http.StatusAccepted, map[string]any{"status": "already-running"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "started"})
	}
}

// kickScan starts a one-off scan in the background using the persisted
// settings (same wiring as the auto-scan loop and the manual "Scan now"
// button). Returns false if a scan is already in flight, in which case
// the caller should not assume a new scan was queued.
//
// Used by /api/scan/run, and by handlers that mutate config (settings
// save, reset) so users see fresh data immediately rather than waiting
// up to one full interval for the auto loop to tick.
func kickScan(st *store.Store) bool {
	if !scanner.TryAcquire() {
		return false
	}
	s := loadSettings(st)
	nets := toScannerNetworks(s)
	var notifyCfg *scanner.NotifyConfig
	if s.SMTP != nil && s.SMTP.Host != "" && len(s.SMTP.Recipients) > 0 {
		notifyCfg = &scanner.NotifyConfig{
			SMTP: scanner.SMTPConfig{
				Host:       s.SMTP.Host,
				Port:       s.SMTP.Port,
				UseTLS:     s.SMTP.UseTLS,
				UseAuth:    s.SMTP.UseAuth,
				Username:   s.SMTP.Username,
				Password:   s.SMTP.Password,
				From:       s.SMTP.From,
				Recipients: s.SMTP.Recipients,
			},
			OnNewHost:    s.Notify.NewHost,
			OnOffline:    s.Notify.Offline,
			OnBackOnline: s.Notify.BackOnline,
		}
	}
	go func() {
		defer scanner.Release()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, _ = scanner.RunOnce(ctx, st, scanner.Settings{
			Networks:         nets,
			ScanIfaces:       s.ScanIfaces,
			ScanEnabled:      true,
			ScanEverySeconds: s.ScanEverySeconds,
			OfflineAfter:     s.OfflineAfter,
			Notify:           notifyCfg,
		})
	}()
	return true
}

func scanStatusHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		running := scanner.IsRunning()
		resp := map[string]any{
			"running": running,
		}
		s := loadSettings(st)
		last, _ := st.GetLastScan()
		if last != nil {
			resp["lastScan"] = last
		}
		// Seconds remaining until the next automatic scan, mirroring
		// scanner.Run's interval logic. Computed against the server's clock
		// (and sent as a delta, not a timestamp) so a client whose clock is
		// skewed renders the same countdown as the scheduler observes.
		if !running && s.ScanEnabled {
			interval := s.ScanEverySeconds
			if interval < 10 {
				interval = 120
			}
			var base int64
			if last != nil && last.EndedAt > 0 {
				base = last.EndedAt
			} else {
				// Fresh DB / post-reset: there's no anchor for "when did the
				// last scan end". The scanner loop wakes up at most every
				// `interval` seconds, so the next scan can be up to a full
				// interval away — that's the only honest countdown we can
				// give without coordinating with the loop. Showing 0 here
				// makes the badge flash "starting…" indefinitely until the
				// loop ticks, which looks broken.
				base = time.Now().Unix()
			}
			remaining := base + int64(interval) - time.Now().Unix()
			if remaining < 0 {
				remaining = 0
			}
			resp["nextScanInSeconds"] = remaining
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// toScannerNetworks projects the stored settings into the scanner's shape.
// Shared by the manual-scan trigger and the scan-targets endpoint so both see
// the same networks.
func toScannerNetworks(s settingsBundle) []scanner.Network {
	out := make([]scanner.Network, 0, len(s.Networks))
	for _, n := range s.Networks {
		var v *int
		if n.VLANID != 0 {
			vid := n.VLANID
			v = &vid
		}
		out = append(out, scanner.Network{Name: n.Name, CIDR: n.CIDR, VLANID: v})
	}
	return out
}

// scanTargetsHandler reports what the next scan will actually probe: one entry
// per interface+CIDR, with the VLAN read off the interface. The Settings page
// renders it so naming a network is a matter of labelling what's there, not
// typing CIDRs from memory.
func scanTargetsHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := loadSettings(st)
		writeJSON(w, http.StatusOK, scanner.Targets(scanner.Settings{
			Networks:   toScannerNetworks(s),
			ScanIfaces: s.ScanIfaces,
		}))
	}
}
