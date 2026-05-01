package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/netglance/netglance/internal/store"
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

func IssueSession(w http.ResponseWriter, st *store.Store, userID int64, secure bool) error {
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
		Secure:   secure,
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
