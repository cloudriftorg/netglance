package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/netglance/netglance/internal/auth"
	"github.com/netglance/netglance/internal/store"
)

// RouterOptions carries cross-handler runtime flags. Currently just Managed,
// which signals that the OPNsense plugin owns part of the settings; the API
// surfaces it via /api/system/managed and rejects writes to those fields.
type RouterOptions struct {
	Managed bool
}

func NewRouter(st *store.Store, webuiHandler http.Handler, opts RouterOptions) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthzHandler(st))

	r.Route("/api", func(r chi.Router) {
		r.Get("/setup/status", setupStatusHandler(st))
		r.Post("/setup", setupHandler(st))
		r.Post("/login", loginHandler(st))
		r.Post("/logout", logoutHandler(st))

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(st))
			r.Get("/me", meHandler(st))

			r.Get("/hosts", listHostsHandler(st))
			r.Get("/hosts/{mac}", getHostHandler(st))
			r.Patch("/hosts/{mac}", updateHostHandler(st))
			r.Delete("/hosts/{mac}", deleteHostHandler(st))
			r.Get("/hosts/{mac}/uptime", uptimeHandler(st))

			r.Post("/scan/run", runScanHandler(st))
			r.Get("/scan/status", scanStatusHandler(st))

			r.Get("/system/interfaces", listInterfacesHandler())
			r.Get("/system/managed", managedHandler(opts.Managed))

			r.Get("/settings", getSettingsHandler(st))
			r.Put("/settings", putSettingsHandler(st, opts.Managed))
			r.Post("/settings/test-smtp", testSMTPHandler(st))

			r.Post("/admin/reset", resetHandler(st))
		})
	})

	r.Handle("/*", webuiHandler)
	return r
}

func healthzHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := st.DB().PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "db-unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func setupStatusHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := st.UserCount()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"setupComplete": n > 0})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
