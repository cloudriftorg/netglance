package store

import "time"

type Scan struct {
	ID         int64  `json:"id"`
	StartedAt  int64  `json:"startedAt"`
	EndedAt    *int64 `json:"endedAt,omitempty"`
	NetworkID  string `json:"networkId,omitempty"`
	HostsFound int    `json:"hostsFound"`
	Error      string `json:"error,omitempty"`
}

func (s *Store) StartScan(networkID string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO scans(started_at, network_id) VALUES(?, ?)`,
		time.Now().Unix(), networkID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishScan(id int64, hostsFound int, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE scans SET ended_at = ?, hosts_found = ?, error = NULLIF(?, '') WHERE id = ?`,
		time.Now().Unix(), hostsFound, errMsg, id,
	)
	return err
}

func (s *Store) ListScans(limit int) ([]Scan, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id, started_at, ended_at, COALESCE(network_id, ''), hosts_found, COALESCE(error, '')
		 FROM scans ORDER BY started_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scan
	for rows.Next() {
		var sc Scan
		var ended *int64
		if err := rows.Scan(&sc.ID, &sc.StartedAt, &ended, &sc.NetworkID, &sc.HostsFound, &sc.Error); err != nil {
			return nil, err
		}
		sc.EndedAt = ended
		out = append(out, sc)
	}
	return out, nil
}
