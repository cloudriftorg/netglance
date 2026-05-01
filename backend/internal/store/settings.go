package store

import (
	"database/sql"
	"encoding/json"
	"errors"
)

func (s *Store) GetSetting(key string, out any) (bool, error) {
	var raw string
	err := s.db.QueryRow(`SELECT v FROM settings WHERE k = ?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SetSetting(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO settings(k, v) VALUES(?, ?)
		ON CONFLICT(k) DO UPDATE SET v = excluded.v
	`, key, string(raw))
	return err
}
