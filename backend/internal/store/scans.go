package store

// LastScan is the single most-recent completed scan record, surfaced in
// the UI as the "Last scan" badge. It's stored under the `lastScan` key
// in the settings table — there's no separate scans history table.
type LastScan struct {
	StartedAt  int64  `json:"startedAt"`
	EndedAt    int64  `json:"endedAt"`
	HostsFound int    `json:"hostsFound"`
	Error      string `json:"error,omitempty"`
}

// GetLastScan returns the most recent completed scan, or nil if none has
// run yet on this instance.
func (s *Store) GetLastScan() (*LastScan, error) {
	var ls *LastScan
	if _, err := s.GetSetting("lastScan", &ls); err != nil {
		return nil, err
	}
	return ls, nil
}

// RecordScan persists the outcome of a completed scan, overwriting the
// previous record.
func (s *Store) RecordScan(ls LastScan) error {
	return s.SetSetting("lastScan", ls)
}
