package api

import (
	"encoding/json"
	"net/http"

	"github.com/cloudriftorg/netglance/internal/auth"
	"github.com/cloudriftorg/netglance/internal/store"
)

type loginRequest struct {
	Password string `json:"password"`
}

func loginHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		u, err := st.Admin()
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
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func logoutHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth.ClearSession(w, r, st)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

type resetRequest struct {
	Password string `json:"password"`
}

// resetHandler wipes the DB and clears the caller's session. Re-authenticates
// the admin password from the body to make sure the action is intentional —
// session-only authorisation would let any leaked cookie nuke the install.
//
// `reBootstrap` is invoked after Reset() to re-apply the env-supplied
// settings (NETGLANCE_SCAN_IFACES, etc). This matters for managed installs
// (OPNsense plugin): without it, scanIfaces stays empty until the daemon
// restarts, and the user can't re-set it from the netglance UI because
// it's owned by the orchestrator.
func resetHandler(st *store.Store, reBootstrap func(*store.Store)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		u, err := st.Admin()
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := auth.CheckPassword(u.PasswordHash, req.Password); err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := st.Reset(); err != nil {
			http.Error(w, "reset failed", http.StatusInternalServerError)
			return
		}
		if reBootstrap != nil {
			reBootstrap(st)
		}
		auth.ClearSession(w, r, st)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func meHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if auth.CurrentUserID(r, st) == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
