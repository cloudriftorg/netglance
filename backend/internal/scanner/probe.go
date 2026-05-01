package scanner

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"
)

const (
	probeWorkers = 32
	probeTimeout = 300 * time.Millisecond
)

var probePorts = []int{80, 443, 22}

func scanNetwork(ctx context.Context, n Network, logger *slog.Logger) []Discovery {
	_, ipnet, err := net.ParseCIDR(n.CIDR)
	if err != nil {
		logger.Warn("invalid cidr", "cidr", n.CIDR, "err", err)
		return nil
	}
	ips := expandCIDR(ipnet)
	if len(ips) == 0 {
		return nil
	}
	jobs := make(chan net.IP, len(ips))
	results := make(chan Discovery, len(ips))
	var wg sync.WaitGroup
	for i := 0; i < probeWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if ctx.Err() != nil {
					return
				}
				if probeAlive(ctx, ip) {
					results <- Discovery{IP: ip, NetworkName: n.Name, VLANID: n.VLANID}
				}
			}
		}()
	}
	for _, ip := range ips {
		jobs <- ip
	}
	close(jobs)
	wg.Wait()
	close(results)
	var out []Discovery
	for r := range results {
		out = append(out, r)
	}
	return out
}

func probeAlive(ctx context.Context, ip net.IP) bool {
	d := net.Dialer{Timeout: probeTimeout}
	for _, p := range probePorts {
		if ctx.Err() != nil {
			return false
		}
		addr := net.JoinHostPort(ip.String(), itoa(p))
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return true
		}
		if isConnRefused(err) {
			return true
		}
	}
	return false
}

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	return containsAny(err.Error(), "connection refused", "refused")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func expandCIDR(ipnet *net.IPNet) []net.IP {
	const maxHosts = 4096
	var out []net.IP
	ones, bits := ipnet.Mask.Size()
	if bits == 32 && ones >= 31 {
		return []net.IP{ipnet.IP.To4()}
	}
	ip := ipnet.IP.Mask(ipnet.Mask).To4()
	if ip == nil {
		return nil
	}
	first := ipv4ToUint32(ip)
	last := first | (^maskToUint32(ipnet.Mask))
	for v := first + 1; v < last && len(out) < maxHosts; v++ {
		out = append(out, uint32ToIPv4(v))
	}
	return out
}

func ipv4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIPv4(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)).To4()
}

func maskToUint32(m net.IPMask) uint32 {
	if len(m) != 4 {
		return 0
	}
	return uint32(m[0])<<24 | uint32(m[1])<<16 | uint32(m[2])<<8 | uint32(m[3])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
