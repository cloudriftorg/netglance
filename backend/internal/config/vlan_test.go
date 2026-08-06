package config

import "testing"

func TestParseIfaceVLANs(t *testing.T) {
	got := ParseIfaceVLANs("vlan02=10, vlan03=20 ,igc1_vlan90=90,broken,bad=x,zero=0,high=4095,=7")
	want := map[string]int{"vlan02": 10, "vlan03": 20, "igc1_vlan90": 90}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %d, want %d", k, got[k], v)
		}
	}
	if len(ParseIfaceVLANs("")) != 0 {
		t.Error("empty input should yield an empty map")
	}
}
