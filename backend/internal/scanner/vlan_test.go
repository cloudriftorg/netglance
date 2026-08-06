package scanner

import "testing"

// The two ways a host gets its VLAN — the supplied map and the tag encoded in
// the device name — plus the one that must never be guessed: OPNsense's bare
// "vlanNN" is a device counter, not a tag.
func TestVLANForIface(t *testing.T) {
	SetIfaceVLANs(map[string]int{"vlan02": 10, "vlan08": 90})
	defer SetIfaceVLANs(nil)

	cases := []struct {
		iface string
		want  int // 0 = expect nil
	}{
		{"vlan02", 10},      // from the map
		{"vlan08", 90},      // from the map
		{"vlan03", 0},       // not in the map, name carries no tag
		{"igc1_vlan20", 20}, // FreeBSD naming
		{"eth0.30", 30},     // Linux naming
		{"vlan0.40", 40},    // OPNsense alternative naming
		{"vlan02x", 0},      // malformed
		{"igc1", 0},         // plain interface
		{"eth0.0", 0},       // 0 is not a valid tag
		{"eth0.4095", 0},    // above the 802.1Q range
	}

	for _, c := range cases {
		got := vlanForIface(c.iface)
		if c.want == 0 {
			if got != nil {
				t.Errorf("%s: got VLAN %d, want none", c.iface, *got)
			}
			continue
		}
		if got == nil {
			t.Errorf("%s: got no VLAN, want %d", c.iface, c.want)
		} else if *got != c.want {
			t.Errorf("%s: got VLAN %d, want %d", c.iface, *got, c.want)
		}
	}
}

// Targets walks a map internally, so without the sort the UI list would
// reshuffle on every reload.
func TestTargetsSortedByVLAN(t *testing.T) {
	tag := func(n int) *int { return &n }
	in := []Target{
		{CIDR: "192.168.90.0/24", VLANID: tag(90)},
		{CIDR: "10.0.0.0/24"},
		{CIDR: "192.168.10.0/24", VLANID: tag(10)},
		{CIDR: "192.168.20.0/24", VLANID: tag(20)},
		{CIDR: "10.1.0.0/24"},
	}
	got := sortTargets(in)

	want := []string{"192.168.10.0/24", "192.168.20.0/24", "192.168.90.0/24", "10.0.0.0/24", "10.1.0.0/24"}
	for i, w := range want {
		if got[i].CIDR != w {
			t.Errorf("position %d: got %s, want %s", i, got[i].CIDR, w)
		}
	}
}
