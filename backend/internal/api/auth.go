package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/netglance/netglance/internal/auth"
	"github.com/netglance/netglance/internal/store"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		u, err := st.UserByUsername(req.Username)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := auth.CheckPassword(u.PasswordHash, req.Password); err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := auth.IssueSession(w, r, st, u.ID); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"username": u.Username})
	}
}

func logoutHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.ClearSession(w, r, st)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func meHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := auth.CurrentUserID(r, st)
		if uid == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var username string
		if err := st.DB().QueryRow(`SELECT username FROM users WHERE id = ?`, uid).Scan(&username); err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"username": username})
	}
}
