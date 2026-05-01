package api

import (
	"context"
	"net/http"
	"time"

	"github.com/netglance/netglance/internal/scanner"
	"github.com/netglance/netglance/internal/store"
)

func runScanHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !scanner.TryAcquire() {
			writeJSON(w, http.StatusAccepted, map[string]any{"status": "already-running"})
			return
		}
		s := loadSettings(st)
		nets := make([]scanner.Network, 0, len(s.Networks))
		for _, n := range s.Networks {
			vid := n.VLANID
			var v *int
			if vid != 0 {
				v = &vid
			}
			nets = append(nets, scanner.Network{Name: n.Name, CIDR: n.CIDR, VLANID: v})
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
			})
		}()
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "started"})
	}
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
				base = time.Now().Unix() - int64(interval)
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
