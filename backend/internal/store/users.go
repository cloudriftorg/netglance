package store

import "time"

func (s *Store) UserCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(username, passwordHash string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users(username, password_hash, created_at) VALUES(?, ?, ?)`,
		username, passwordHash, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

func (s *Store) UserByUsername(username string) (*User, error) {
	row := s.db.QueryRow(`SELECT id, username, password_hash FROM users WHERE username = ?`, username)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}
