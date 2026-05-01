package main

import (
	"context"
	"errors"
	"fmt"
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

// version is overridden at build time via -ldflags="-X main.version=..."
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(runHealthcheck())
		case "version", "--version", "-v":
			fmt.Println(version)
			return
		}
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
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
		slog.Info("netglance starting", "version", version, "bind", cfg.Bind, "data", cfg.DataDir)
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

// runHealthcheck is invoked by `netglance healthcheck` from the container's
// HEALTHCHECK directive — distroless has no shell or curl, so the binary
// dials its own /healthz endpoint and exits 0/1 accordingly.
func runHealthcheck() int {
	addr := os.Getenv("NETGLANCE_BIND")
	if addr == "" || addr == ":8080" {
		addr = "127.0.0.1:8080"
	} else if addr[0] == ':' {
		addr = "127.0.0.1" + addr
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

type netCfg struct {
	Name   string `json:"name"`
	CIDR   string `json:"cidr"`
	VLANID int    `json:"vlanId,omitempty"`
}

func loadScannerSettings(st *store.Store) scanner.Settings {
	var nets []netCfg
	_, _ = st.GetSetting("networks", &nets)
	out := scanner.Settings{ScanEnabled: true, ScanEverySeconds: 120, OfflineAfter: 1}
	_, _ = st.GetSetting("scanEnabled", &out.ScanEnabled)
	_, _ = st.GetSetting("scanEverySeconds", &out.ScanEverySeconds)
	_, _ = st.GetSetting("offlineAfter", &out.OfflineAfter)
	_, _ = st.GetSetting("scanIfaces", &out.ScanIfaces)
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
