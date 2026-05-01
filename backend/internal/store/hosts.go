package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Host struct {
	ID            int64  `json:"id"`
	MAC           string `json:"mac"`
	IP            string `json:"ip"`
	VLANID        *int   `json:"vlanId,omitempty"`
	NetworkName   string `json:"networkName,omitempty"`
	Hostname      string `json:"hostname,omitempty"`
	Vendor        string `json:"vendor,omitempty"`
	CustomVendor  string `json:"customVendor,omitempty"`
	CustomName    string `json:"customName,omitempty"`
	FirstSeen     int64  `json:"firstSeen"`
	LastSeen      int64  `json:"lastSeen"`
	Online        bool   `json:"online"`
	IsNew         bool   `json:"isNew"`
	NotifyOffline bool   `json:"notifyOffline"`
}

type HostFilter struct {
	VLAN   *int
	Online *bool
	Query  string
}

func (s *Store) UpsertSeen(mac, ip, networkName string, vlanID *int, vendor, hostname string, ts int64) (*Host, bool, error) {
	mac = strings.ToLower(mac)
	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var existing Host
	var v sql.NullInt64
	err = tx.QueryRow(
		`SELECT id, mac, ip, vlan_id, COALESCE(network_name,''), COALESCE(hostname,''),
		        COALESCE(vendor,''), COALESCE(custom_vendor,''), COALESCE(custom_name,''),
		        first_seen, last_seen, online, is_new, notify_offline
		 FROM hosts WHERE mac = ?`, mac,
	).Scan(
		&existing.ID, &existing.MAC, &existing.IP, &v, &existing.NetworkName, &existing.Hostname,
		&existing.Vendor, &existing.CustomVendor, &existing.CustomName,
		&existing.FirstSeen, &existing.LastSeen, &existing.Online, &existing.IsNew, &existing.NotifyOffline,
	)
	if v.Valid {
		i := int(v.Int64)
		existing.VLANID = &i
	}

	isNew := errors.Is(err, sql.ErrNoRows)
	if err != nil && !isNew {
		return nil, false, err
	}

	if isNew {
		var vid any
		if vlanID != nil {
			vid = *vlanID
		}
		res, err := tx.Exec(
			`INSERT INTO hosts(mac, ip, vlan_id, network_name, hostname, vendor, first_seen, last_seen, online, missed_scans, is_new)
			 VALUES(?, ?, ?, ?, ?, ?, ?, ?, 1, 0, 1)`,
			mac, ip, vid, networkName, hostname, vendor, ts, ts,
		)
		if err != nil {
			return nil, false, err
		}
		id, _ := res.LastInsertId()
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'new', ?)`, id, ts, ip); err != nil {
			return nil, false, err
		}
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'online', ?)`, id, ts, ip); err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		h, _ := s.HostByMAC(mac)
		return h, true, nil
	}

	wasOffline := !existing.Online
	ipChanged := existing.IP != ip
	var vid any
	if vlanID != nil {
		vid = *vlanID
	}
	if _, err := tx.Exec(
		`UPDATE hosts
		 SET ip = ?, vlan_id = COALESCE(?, vlan_id), network_name = COALESCE(NULLIF(?, ''), network_name),
		     hostname = COALESCE(NULLIF(?, ''), hostname),
		     vendor   = COALESCE(NULLIF(?, ''), vendor),
		     last_seen = ?, online = 1, missed_scans = 0
		 WHERE id = ?`,
		ip, vid, networkName, hostname, vendor, ts, existing.ID,
	); err != nil {
		return nil, false, err
	}
	if wasOffline {
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'online', ?)`, existing.ID, ts, ip); err != nil {
			return nil, false, err
		}
	}
	if ipChanged {
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'ip_change', ?)`, existing.ID, ts, ip); err != nil {
			return nil, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	h, _ := s.HostByMAC(mac)
	return h, false, nil
}

func (s *Store) MarkSweep(seenSince int64, offlineThreshold int) ([]*Host, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE hosts SET missed_scans = missed_scans + 1 WHERE last_seen < ? AND online = 1`,
		seenSince,
	); err != nil {
		return nil, err
	}
	rows, err := tx.Query(`SELECT id, mac FROM hosts WHERE online = 1 AND missed_scans >= ?`, offlineThreshold)
	if err != nil {
		return nil, err
	}
	type row struct {
		id  int64
		mac string
	}
	var toOffline []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.mac); err != nil {
			rows.Close()
			return nil, err
		}
		toOffline = append(toOffline, r)
	}
	rows.Close()
	now := time.Now().Unix()
	for _, r := range toOffline {
		if _, err := tx.Exec(`UPDATE hosts SET online = 0 WHERE id = ?`, r.id); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'offline', NULL)`, r.id, now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := make([]*Host, 0, len(toOffline))
	for _, r := range toOffline {
		if h, err := s.HostByMAC(r.mac); err == nil {
			out = append(out, h)
		}
	}
	return out, nil
}

func (s *Store) HostByMAC(mac string) (*Host, error) {
	mac = strings.ToLower(mac)
	var h Host
	var v sql.NullInt64
	err := s.db.QueryRow(
		`SELECT id, mac, ip, vlan_id, COALESCE(network_name,''), COALESCE(hostname,''),
		        COALESCE(vendor,''), COALESCE(custom_vendor,''), COALESCE(custom_name,''),
		        first_seen, last_seen, online, is_new, notify_offline
		 FROM hosts WHERE mac = ?`, mac,
	).Scan(
		&h.ID, &h.MAC, &h.IP, &v, &h.NetworkName, &h.Hostname,
		&h.Vendor, &h.CustomVendor, &h.CustomName, &h.FirstSeen, &h.LastSeen, &h.Online, &h.IsNew, &h.NotifyOffline,
	)
	if err != nil {
		return nil, err
	}
	if v.Valid {
		i := int(v.Int64)
		h.VLANID = &i
	}
	return &h, nil
}

func (s *Store) ListHosts(f HostFilter) ([]*Host, error) {
	q := strings.Builder{}
	q.WriteString(`SELECT id, mac, ip, vlan_id, COALESCE(network_name,''), COALESCE(hostname,''),
	                       COALESCE(vendor,''), COALESCE(custom_vendor,''), COALESCE(custom_name,''),
	                       first_seen, last_seen, online, is_new, notify_offline
	               FROM hosts WHERE 1=1`)
	args := []any{}
	if f.VLAN != nil {
		q.WriteString(` AND vlan_id = ?`)
		args = append(args, *f.VLAN)
	}
	if f.Online != nil {
		q.WriteString(` AND online = ?`)
		v := 0
		if *f.Online {
			v = 1
		}
		args = append(args, v)
	}
	if f.Query != "" {
		q.WriteString(` AND (mac LIKE ? OR ip LIKE ? OR hostname LIKE ? OR custom_name LIKE ?)`)
		needle := "%" + strings.ToLower(f.Query) + "%"
		args = append(args, needle, needle, needle, needle)
	}
	q.WriteString(` ORDER BY online DESC, last_seen DESC LIMIT 1000`)
	rows, err := s.db.Query(q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Host
	for rows.Next() {
		var h Host
		var v sql.NullInt64
		if err := rows.Scan(
			&h.ID, &h.MAC, &h.IP, &v, &h.NetworkName, &h.Hostname,
			&h.Vendor, &h.CustomVendor, &h.CustomName, &h.FirstSeen, &h.LastSeen, &h.Online, &h.IsNew, &h.NotifyOffline,
		); err != nil {
			return nil, err
		}
		if v.Valid {
			i := int(v.Int64)
			h.VLANID = &i
		}
		out = append(out, &h)
	}
	return out, nil
}

func (s *Store) UpdateHostMeta(mac, customName, customVendor string, notifyOffline, isNew bool) error {
	mac = strings.ToLower(mac)
	_, err := s.db.Exec(
		`UPDATE hosts SET custom_name = ?, custom_vendor = ?, notify_offline = ?, is_new = ? WHERE mac = ?`,
		customName, customVendor, boolToInt(notifyOffline), boolToInt(isNew), mac,
	)
	return err
}

type HostEvent struct {
	ID     int64  `json:"id"`
	HostID int64  `json:"hostId"`
	TS     int64  `json:"ts"`
	Kind   string `json:"kind"`
	IP     string `json:"ip,omitempty"`
}

func (s *Store) HostEvents(hostID int64, since int64, limit int) ([]HostEvent, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT id, host_id, ts, kind, COALESCE(ip,'') FROM host_events
		 WHERE host_id = ? AND ts >= ? ORDER BY ts DESC LIMIT ?`,
		hostID, since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HostEvent
	for rows.Next() {
		var e HostEvent
		if err := rows.Scan(&e.ID, &e.HostID, &e.TS, &e.Kind, &e.IP); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
