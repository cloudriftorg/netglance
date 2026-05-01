package store

import "time"

// Netglance is single-user. The schema keeps a `username` column for
// historical reasons but the application only ever has one row, identified
// internally by the constant returned from auth/setup.
const adminUsername = "admin"

func (s *Store) UserCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateAdmin(passwordHash string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users(username, password_hash, created_at) VALUES(?, ?, ?)`,
		adminUsername, passwordHash, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

type User struct {
	ID           int64
	PasswordHash string
}

// Reset wipes every row from every application table. The schema (and the
// active connection) is preserved so the running process keeps working — the
// next request hits an empty DB exactly as if it had just been provisioned,
// triggering the setup wizard. Sessions are cleared, so the caller is
// effectively logged out.
func (s *Store) Reset() error {
	tables := []string{"host_events", "hosts", "sessions", "users", "settings"}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range tables {
		if _, err := tx.Exec("DELETE FROM " + t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Admin() (*User, error) {
	row := s.db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, adminUsername)
	var u User
	if err := row.Scan(&u.ID, &u.PasswordHash); err != nil {
		return nil, err
	}
	return &u, nil
}
