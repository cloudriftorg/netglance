package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/netglance/netglance/internal/auth"
	"github.com/netglance/netglance/internal/store"
)

type setupRequest struct {
	Username  string          `json:"username"`
	Password  string          `json:"password"`
	Networks  []NetworkConfig `json:"networks,omitempty"`
	SMTP      *SMTPConfig     `json:"smtp,omitempty"`
	ScanEvery int             `json:"scanEverySeconds,omitempty"`
}

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
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		uid, err := st.CreateUser(req.Username, hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(req.Networks) > 0 {
			_ = st.SetSetting("networks", req.Networks)
		}
		if req.SMTP != nil {
			_ = st.SetSetting("smtp", req.SMTP)
		}
		if req.ScanEvery > 0 {
			_ = st.SetSetting("scanEverySeconds", req.ScanEvery)
		}
		if err := auth.IssueSession(w, r, st, uid); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"username": req.Username})
	}
}
