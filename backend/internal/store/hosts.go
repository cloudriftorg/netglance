package store

import (
	"database/sql"
	"errors"
	"net"
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
	NotifyOnline  bool   `json:"notifyOnline"`
}

type HostFilter struct {
	VLAN   *int
	Online *bool
	Query  string
}

// UpsertSeen returns:
//   host       — current row after the upsert
//   isNew      — true on first-ever sight of this MAC
//   wasOffline — true when this scan flipped an existing host from
//                offline back to online (so the caller can fire a
//                'back online' notification)
func (s *Store) UpsertSeen(mac, ip, networkName string, vlanID *int, vendor, hostname string, ts int64) (*Host, bool, bool, error) {
	mac = strings.ToLower(mac)
	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, false, err
	}
	defer tx.Rollback()

	var existing Host
	var v sql.NullInt64
	err = tx.QueryRow(
		`SELECT id, mac, ip, vlan_id, COALESCE(network_name,''), COALESCE(hostname,''),
		        COALESCE(vendor,''), COALESCE(custom_vendor,''), COALESCE(custom_name,''),
		        first_seen, last_seen, online, is_new, notify_offline, notify_online
		 FROM hosts WHERE mac = ?`, mac,
	).Scan(
		&existing.ID, &existing.MAC, &existing.IP, &v, &existing.NetworkName, &existing.Hostname,
		&existing.Vendor, &existing.CustomVendor, &existing.CustomName,
		&existing.FirstSeen, &existing.LastSeen, &existing.Online, &existing.IsNew, &existing.NotifyOffline, &existing.NotifyOnline,
	)
	if v.Valid {
		i := int(v.Int64)
		existing.VLANID = &i
	}

	isNew := errors.Is(err, sql.ErrNoRows)
	if err != nil && !isNew {
		return nil, false, false, err
	}

	if isNew {
		var vid any
		if vlanID != nil {
			vid = *vlanID
		}
		// notify_offline / notify_online forced to 0 on insert: notifications
		// are opt-in per host. The user enables them from the host detail
		// page, gated by the global toggles in Settings.
		res, err := tx.Exec(
			`INSERT INTO hosts(mac, ip, vlan_id, network_name, hostname, vendor, first_seen, last_seen, online, missed_scans, is_new, notify_offline, notify_online)
			 VALUES(?, ?, ?, ?, ?, ?, ?, ?, 1, 0, 1, 0, 0)`,
			mac, ip, vid, networkName, hostname, vendor, ts, ts,
		)
		if err != nil {
			return nil, false, false, err
		}
		id, _ := res.LastInsertId()
		// Only the 'new' event is recorded on first sight — a "Came online"
		// at the same instant would be redundant ("First seen" already
		// implies the host is online). 'online' events are reserved for
		// later offline→online transitions.
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'new', ?)`, id, ts, ip); err != nil {
			return nil, false, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, false, err
		}
		h, _ := s.HostByMAC(mac)
		return h, true, false, nil
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
		return nil, false, false, err
	}
	if wasOffline {
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'online', ?)`, existing.ID, ts, ip); err != nil {
			return nil, false, false, err
		}
	}
	if ipChanged {
		if _, err := tx.Exec(`INSERT INTO host_events(host_id, ts, kind, ip) VALUES(?, ?, 'ip_change', ?)`, existing.ID, ts, ip); err != nil {
			return nil, false, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, false, false, err
	}
	h, _ := s.HostByMAC(mac)
	return h, false, wasOffline, nil
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
		        first_seen, last_seen, online, is_new, notify_offline, notify_online
		 FROM hosts WHERE mac = ?`, mac,
	).Scan(
		&h.ID, &h.MAC, &h.IP, &v, &h.NetworkName, &h.Hostname,
		&h.Vendor, &h.CustomVendor, &h.CustomName, &h.FirstSeen, &h.LastSeen, &h.Online, &h.IsNew, &h.NotifyOffline, &h.NotifyOnline,
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
	                       first_seen, last_seen, online, is_new, notify_offline, notify_online
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
			&h.Vendor, &h.CustomVendor, &h.CustomName, &h.FirstSeen, &h.LastSeen, &h.Online, &h.IsNew, &h.NotifyOffline, &h.NotifyOnline,
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

// DeleteHost removes a host (and via FK cascade, all its host_events).
// The next scan that finds the same MAC re-inserts the host as brand-new
// (is_new = 1), so deletion is the supported way to reset a host to its
// NEW state.
func (s *Store) DeleteHost(mac string) error {
	mac = strings.ToLower(mac)
	_, err := s.db.Exec(`DELETE FROM hosts WHERE mac = ?`, mac)
	return err
}

// DeleteAllHosts wipes the entire host inventory. host_events cascade via
// FK. Used by the "Clear list" action — a fresh scan will re-inventory
// everything as brand-new.
func (s *Store) DeleteAllHosts() error {
	_, err := s.db.Exec(`DELETE FROM hosts`)
	return err
}

// RetagHosts re-evaluates each host's vlan_id and network_name against the
// supplied network list, matching by CIDR containment. Run this after the
// user saves network changes in Settings so a VLAN rename/retag reflects
// on existing hosts immediately, without forcing a full rescan. Hosts that
// don't match any configured CIDR are cleared (vlan_id = NULL).
func (s *Store) RetagHosts(networks []NetworkRule) error {
	rows, err := s.db.Query(`SELECT id, ip FROM hosts`)
	if err != nil {
		return err
	}
	type row struct {
		id int64
		ip string
	}
	var hosts []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ip); err != nil {
			rows.Close()
			return err
		}
		hosts = append(hosts, r)
	}
	rows.Close()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, h := range hosts {
		var vlanID *int
		var name string
		for _, n := range networks {
			if n.Contains(h.ip) {
				if n.VLANID != 0 {
					v := n.VLANID
					vlanID = &v
				}
				name = n.Name
				break
			}
		}
		if _, err := tx.Exec(
			`UPDATE hosts SET vlan_id = ?, network_name = ? WHERE id = ?`,
			vlanID, name, h.id,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// NetworkRule is a minimal projection of the user-configured Networks slice
// passed to RetagHosts. The store package owns the IP-in-CIDR check so the
// settings handler doesn't have to import net.
type NetworkRule struct {
	CIDR   string
	VLANID int
	Name   string
}

// Contains reports whether ip is inside the rule's CIDR. Empty/invalid
// inputs return false.
func (r NetworkRule) Contains(ip string) bool {
	if r.CIDR == "" || ip == "" {
		return false
	}
	_, ipnet, err := net.ParseCIDR(r.CIDR)
	if err != nil {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return ipnet.Contains(parsed)
}

func (s *Store) UpdateHostMeta(mac, customName, customVendor string, notifyOffline, notifyOnline, isNew bool) error {
	mac = strings.ToLower(mac)
	_, err := s.db.Exec(
		`UPDATE hosts SET custom_name = ?, custom_vendor = ?, notify_offline = ?, notify_online = ?, is_new = ? WHERE mac = ?`,
		customName, customVendor, boolToInt(notifyOffline), boolToInt(notifyOnline), boolToInt(isNew), mac,
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
