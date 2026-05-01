package api

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/netglance/netglance/internal/scanner"
	"github.com/netglance/netglance/internal/store"
)

var scanInFlight int32

func runScanHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !atomic.CompareAndSwapInt32(&scanInFlight, 0, 1) {
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
			defer atomic.StoreInt32(&scanInFlight, 0)
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

func scanStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"running": atomic.LoadInt32(&scanInFlight) == 1,
		})
	}
}
