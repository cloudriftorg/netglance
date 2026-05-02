package api

import (
	"encoding/json"
	"net/http"

	"github.com/netglance/netglance/internal/store"
)

type settingsBundle struct {
	Networks         []NetworkConfig `json:"networks"`
	ScanEnabled      bool            `json:"scanEnabled"`
	ScanEverySeconds int             `json:"scanEverySeconds"`
	ScanIfaces       []string        `json:"scanIfaces"`
	OfflineAfter     int             `json:"offlineAfter"`
	SMTP             *SMTPConfig     `json:"smtp,omitempty"`
	Notify           NotifyToggles   `json:"notify"`
}

type NotifyToggles struct {
	NewHost    bool `json:"newHost"`
	Offline    bool `json:"offline"`
	BackOnline bool `json:"backOnline"`
}

func defaultSettings() settingsBundle {
	return settingsBundle{
		Networks:         []NetworkConfig{},
		ScanEnabled:      true,
		ScanEverySeconds: 120,
		ScanIfaces:       []string{},
		OfflineAfter:     1,
		Notify:           NotifyToggles{NewHost: true, Offline: true, BackOnline: false},
	}
}

func getSettingsHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := loadSettings(st)
		if s.SMTP != nil && s.SMTP.Password != "" {
			s.SMTP.Password = "********"
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func putSettingsHandler(st *store.Store, managed bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req settingsBundle
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		current := loadSettings(st)
		if req.SMTP != nil && req.SMTP.Password == "********" && current.SMTP != nil {
			req.SMTP.Password = current.SMTP.Password
		}
		// In managed mode (OPNsense plugin), revert writes to fields that the
		// orchestrator owns — the OPNsense plugin rewrites them on every
		// service start anyway, so a UI edit would silently disappear. The
		// frontend renders the same fields read-only via /api/system/managed;
		// this is the server-side guard. Mirror config.ManagedFields exactly.
		if managed {
			req.ScanIfaces = current.ScanIfaces
		}
		_ = st.SetSetting("networks", req.Networks)
		_ = st.SetSetting("scanEnabled", req.ScanEnabled)
		_ = st.SetSetting("scanEverySeconds", req.ScanEverySeconds)
		_ = st.SetSetting("scanIfaces", req.ScanIfaces)
		_ = st.SetSetting("offlineAfter", req.OfflineAfter)
		_ = st.SetSetting("smtp", req.SMTP)
		_ = st.SetSetting("notify", req.Notify)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func loadSettings(st *store.Store) settingsBundle {
	s := defaultSettings()
	_, _ = st.GetSetting("networks", &s.Networks)
	_, _ = st.GetSetting("scanEnabled", &s.ScanEnabled)
	_, _ = st.GetSetting("scanEverySeconds", &s.ScanEverySeconds)
	_, _ = st.GetSetting("scanIfaces", &s.ScanIfaces)
	_, _ = st.GetSetting("offlineAfter", &s.OfflineAfter)
	_, _ = st.GetSetting("smtp", &s.SMTP)
	_, _ = st.GetSetting("notify", &s.Notify)
	return s
}
