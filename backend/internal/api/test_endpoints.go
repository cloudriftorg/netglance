package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudriftorg/netglance/internal/notify"
	"github.com/cloudriftorg/netglance/internal/store"
)

func testSMTPHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := loadSettings(st)
		if s.SMTP == nil {
			http.Error(w, "smtp not configured", http.StatusBadRequest)
			return
		}
		err := notify.Send(notify.Config{
			Host:       s.SMTP.Host,
			Port:       s.SMTP.Port,
			UseTLS:     s.SMTP.UseTLS,
			UseAuth:    s.SMTP.UseAuth,
			Username:   s.SMTP.Username,
			Password:   s.SMTP.Password,
			From:       s.SMTP.From,
			Recipients: s.SMTP.Recipients,
		}, "Netglance — test email", fmt.Sprintf("Test sent at %s", time.Now().Format(time.RFC1123)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
