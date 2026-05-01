package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/netglance/netglance/internal/store"
)

func listHostsHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f := store.HostFilter{Query: q.Get("q")}
		if v := q.Get("vlan"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				f.VLAN = &n
			}
		}
		if v := q.Get("online"); v != "" {
			b := v == "1" || v == "true"
			f.Online = &b
		}
		hosts, err := st.ListHosts(f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, hosts)
	}
}

func getHostHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mac := chi.URLParam(r, "mac")
		h, err := st.HostByMAC(mac)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		since := time.Now().Add(-7 * 24 * time.Hour).Unix()
		events, err := st.HostEvents(h.ID, since, 200)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"host": h, "events": events})
	}
}

type updateHostRequest struct {
	CustomName    string `json:"customName"`
	CustomVendor  string `json:"customVendor"`
	NotifyOffline bool   `json:"notifyOffline"`
	IsNew         bool   `json:"isNew"`
}

func updateHostHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mac := chi.URLParam(r, "mac")
		var req updateHostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if err := st.UpdateHostMeta(mac, req.CustomName, req.CustomVendor, req.NotifyOffline, req.IsNew); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h, err := st.HostByMAC(mac)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, h)
	}
}

func deleteHostHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mac := chi.URLParam(r, "mac")
		if err := st.DeleteHost(mac); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func uptimeHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mac := chi.URLParam(r, "mac")
		h, err := st.HostByMAC(mac)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		rangeStr := r.URL.Query().Get("range")
		var dur time.Duration
		switch rangeStr {
		case "24h", "":
			dur = 24 * time.Hour
		case "7d":
			dur = 7 * 24 * time.Hour
		case "30d":
			dur = 30 * 24 * time.Hour
		default:
			http.Error(w, "invalid range", http.StatusBadRequest)
			return
		}
		since := time.Now().Add(-dur).Unix()
		events, err := st.HostEvents(h.ID, since, 5000)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"host": h, "since": since, "events": events})
	}
}

