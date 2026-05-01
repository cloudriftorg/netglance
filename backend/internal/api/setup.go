package api

import (
	"encoding/json"
	"net/http"

	"github.com/netglance/netglance/internal/auth"
	"github.com/netglance/netglance/internal/store"
)

type NetworkConfig struct {
	Name   string `json:"name"`
	CIDR   string `json:"cidr"`
	VLANID int    `json:"vlanId,omitempty"`
}

type SMTPConfig struct {
	Host       string   `json:"host"`
	Port       int      `json:"port"`
	UseTLS     bool     `json:"useTLS"`
	UseAuth    bool     `json:"useAuth"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	From       string   `json:"from"`
	Recipients []string `json:"recipients"`
}

// setupRequest is the first wizard step: just the admin password. Interface
// selection is a separate authenticated step done after this succeeds, so the
// listInterfaces endpoint stays uniformly auth-gated.
type setupRequest struct {
	Password string `json:"password"`
}

func setupHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := st.UserCount()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if n > 0 {
			http.Error(w, "setup already completed", http.StatusConflict)
			return
		}
		var req setupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		uid, err := st.CreateAdmin(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := auth.IssueSession(w, r, st, uid); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	}
}
