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
		resp := map[string]any{
			"running": scanner.IsRunning(),
		}
		if last, err := st.GetLastScan(); err == nil && last != nil {
			resp["lastScan"] = last
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
