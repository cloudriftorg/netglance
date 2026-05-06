package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cloudriftorg/netglance/internal/store"
)

const (
	cookieName   = "netglance_sid"
	sessionTTL   = 30 * 24 * time.Hour
	bcryptCost   = 12
	tokenByteLen = 32
)

func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", errors.New("password too short (min 8)")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// isHTTPS returns true when the request is coming over HTTPS, either directly
// (TLS terminated by us) or via a trusted reverse proxy that set
// X-Forwarded-Proto. Used to gate the cookie Secure flag — without proxy
// awareness, a TLS-terminated reverse proxy + plain backend would never get
// a Secure cookie even though the user-facing connection is HTTPS.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		// May contain a comma-separated list when traffic crosses multiple
		// proxies — the first value is the closest to the client.
		first := strings.TrimSpace(strings.SplitN(proto, ",", 2)[0])
		if strings.EqualFold(first, "https") {
			return true
		}
	}
	return false
}

func IssueSession(w http.ResponseWriter, r *http.Request, st *store.Store, userID int64) error {
	tok, err := randomToken()
	if err != nil {
		return err
	}
	now := time.Now()
	exp := now.Add(sessionTTL)
	if _, err := st.DB().Exec(
		`INSERT INTO sessions(token, user_id, created_at, expires_at) VALUES(?, ?, ?, ?)`,
		tok, userID, now.Unix(), exp.Unix(),
	); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
	return nil
}

func ClearSession(w http.ResponseWriter, r *http.Request, st *store.Store) {
	if c, err := r.Cookie(cookieName); err == nil {
		_, _ = st.DB().Exec(`DELETE FROM sessions WHERE token = ?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
}

func CurrentUserID(r *http.Request, st *store.Store) int64 {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return 0
	}
	var uid, exp int64
	err = st.DB().QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token = ?`, c.Value).Scan(&uid, &exp)
	if err != nil {
		return 0
	}
	if time.Now().Unix() > exp {
		_, _ = st.DB().Exec(`DELETE FROM sessions WHERE token = ?`, c.Value)
		return 0
	}
	return uid
}

func RequireAuth(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if CurrentUserID(r, st) == 0 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func randomToken() (string, error) {
	b := make([]byte, tokenByteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
