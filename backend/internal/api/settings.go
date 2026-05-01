package api

import (
	"encoding/json"
	"net/http"

	"github.com/netglance/netglance/internal/store"
)

type settingsBundle struct {
	Networks         []NetworkConfig `json:"networks"`
	ScanEverySeconds int             `json:"scanEverySeconds"`
	OfflineAfter     int             `json:"offlineAfter"`
	PrimaryIface     string          `json:"primaryIface"`
	SMTP             *SMTPConfig     `json:"smtp,omitempty"`
	Gateway          *GatewayConfig  `json:"gateway,omitempty"`
	Notify           NotifyToggles   `json:"notify"`
}

type GatewayConfig struct {
	Adapter   string `json:"adapter"`
	URL       string `json:"url"`
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
	VerifyTLS bool   `json:"verifyTLS"`
}

type NotifyToggles struct {
	NewHost    bool `json:"newHost"`
	Offline    bool `json:"offline"`
	BackOnline bool `json:"backOnline"`
}

func defaultSettings() settingsBundle {
	return settingsBundle{
		Networks:         []NetworkConfig{},
		ScanEverySeconds: 300,
		OfflineAfter:     3,
		Notify:           NotifyToggles{NewHost: true, Offline: true, BackOnline: false},
	}
}

func getSettingsHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := loadSettings(st)
		if s.SMTP != nil && s.SMTP.Password != "" {
			s.SMTP.Password = "********"
		}
		if s.Gateway != nil && s.Gateway.APISecret != "" {
			s.Gateway.APISecret = "********"
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func putSettingsHandler(st *store.Store) http.HandlerFunc {
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
		if req.Gateway != nil && req.Gateway.APISecret == "********" && current.Gateway != nil {
			req.Gateway.APISecret = current.Gateway.APISecret
		}
		_ = st.SetSetting("networks", req.Networks)
		_ = st.SetSetting("scanEverySeconds", req.ScanEverySeconds)
		_ = st.SetSetting("offlineAfter", req.OfflineAfter)
		_ = st.SetSetting("primaryIface", req.PrimaryIface)
		_ = st.SetSetting("smtp", req.SMTP)
		_ = st.SetSetting("gateway", req.Gateway)
		_ = st.SetSetting("notify", req.Notify)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func loadSettings(st *store.Store) settingsBundle {
	s := defaultSettings()
	_, _ = st.GetSetting("networks", &s.Networks)
	_, _ = st.GetSetting("scanEverySeconds", &s.ScanEverySeconds)
	_, _ = st.GetSetting("offlineAfter", &s.OfflineAfter)
	_, _ = st.GetSetting("primaryIface", &s.PrimaryIface)
	_, _ = st.GetSetting("smtp", &s.SMTP)
	_, _ = st.GetSetting("gateway", &s.Gateway)
	_, _ = st.GetSetting("notify", &s.Notify)
	return s
}
