package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netglance/netglance/internal/api"
	"github.com/netglance/netglance/internal/config"
	"github.com/netglance/netglance/internal/scanner"
	"github.com/netglance/netglance/internal/store"
	"github.com/netglance/netglance/internal/webui"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		slog.Error("create data dir", "err", err)
		os.Exit(1)
	}

	st, err := store.Open(cfg.DataDir + "/netglance.db")
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		for ctx.Err() == nil {
			if n, _ := st.UserCount(); n > 0 {
				break
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
		scanner.Run(ctx, st, func() scanner.Settings {
			return loadScannerSettings(st)
		})
	}()

	router := api.NewRouter(st, webui.Handler())
	srv := &http.Server{
		Addr:              cfg.Bind,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("netglance starting", "bind", cfg.Bind, "data", cfg.DataDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

type netCfg struct {
	Name   string `json:"name"`
	CIDR   string `json:"cidr"`
	VLANID int    `json:"vlanId,omitempty"`
}

func loadScannerSettings(st *store.Store) scanner.Settings {
	var nets []netCfg
	_, _ = st.GetSetting("networks", &nets)
	out := scanner.Settings{ScanEverySeconds: 300, OfflineAfter: 3}
	_, _ = st.GetSetting("scanEverySeconds", &out.ScanEverySeconds)
	_, _ = st.GetSetting("offlineAfter", &out.OfflineAfter)
	_, _ = st.GetSetting("primaryIface", &out.PrimaryIface)
	for _, n := range nets {
		var v *int
		if n.VLANID != 0 {
			vid := n.VLANID
			v = &vid
		}
		out.Networks = append(out.Networks, scanner.Network{Name: n.Name, CIDR: n.CIDR, VLANID: v})
	}
	return out
}
