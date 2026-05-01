package ouidb

import "strings"

var builtin = map[string]string{
	"525400": "QEMU/KVM",
	"BC2411": "Proxmox VE",
	"080027": "VirtualBox",
	"005056": "VMware",
	"00163E": "Xensource",
	"001C42": "Parallels",
	"A4C361": "Apple",
	"3C2EFF": "Apple",
	"B827EB": "Raspberry Pi",
	"DCA632": "Raspberry Pi",
	"E45F01": "Raspberry Pi",
	"00156D": "Ubiquiti",
	"24A43C": "Ubiquiti",
	"FCECDA": "Ubiquiti",
	"E063DA": "Ubiquiti",
	"00904C": "Epigram",
	"DCFE07": "Tenda",
	"C83A35": "Tenda",
	"50C7BF": "TP-Link",
	"E848B8": "TP-Link",
	"AC84C6": "TP-Link",
	"002566": "Shelly / Espressif",
	"BCFF4D": "ASUSTek",
}

func Lookup(mac string) string {
	prefix := normalize(mac)
	if prefix == "" {
		return ""
	}
	if v, ok := builtin[prefix]; ok {
		return v
	}
	return ""
}

func normalize(mac string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
			return r
		}
		return -1
	}, mac)
	if len(clean) < 6 {
		return ""
	}
	return strings.ToUpper(clean[:6])
}
