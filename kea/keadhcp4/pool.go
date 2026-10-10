package keadhcp4

import (
	"fmt"
	"net/netip"
	"strings"
)

// ParsePool returns the first and last address of a pool written the way
// Kea accepts it: a range "192.0.2.10 - 192.0.2.20" or a prefix
// "192.0.2.64/26" without host bits. Kea reports pools in its own spelling,
// so pools are compared by their range.
func ParsePool(s string) (first, last netip.Addr, err error) {
	if before, after, ok := strings.Cut(s, "/"); ok {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(before) + "/" + strings.TrimSpace(after))
		if err != nil {
			return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: %w", s, err)
		}
		if !prefix.Addr().Is4() {
			return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: not an IPv4 prefix", s)
		}
		if prefix.Masked() != prefix {
			return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: prefix has host bits set", s)
		}
		return prefix.Addr(), lastAddr(prefix), nil
	}

	before, after, ok := strings.Cut(s, "-")
	if !ok {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: expected a range \"first - last\" or a prefix", s)
	}
	first, err = netip.ParseAddr(strings.TrimSpace(before))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: %w", s, err)
	}
	last, err = netip.ParseAddr(strings.TrimSpace(after))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: %w", s, err)
	}
	if !first.Is4() || !last.Is4() {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: not an IPv4 range", s)
	}
	if last.Less(first) {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("pool %q: last address is lower than first address", s)
	}
	return first, last, nil
}

func lastAddr(prefix netip.Prefix) netip.Addr {
	a := prefix.Addr().As4()
	for i := prefix.Bits(); i < 32; i++ {
		a[i/8] |= 1 << (7 - i%8)
	}
	return netip.AddrFrom4(a)
}
