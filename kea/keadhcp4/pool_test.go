package keadhcp4

import (
	"net/netip"
	"testing"
)

func TestParsePool(t *testing.T) {
	tests := []struct {
		in          string
		first, last string
	}{
		{"10.0.0.10 - 10.0.0.20", "10.0.0.10", "10.0.0.20"},
		{"10.0.0.10-10.0.0.20", "10.0.0.10", "10.0.0.20"},
		{" 10.0.0.10   -10.0.0.20 ", "10.0.0.10", "10.0.0.20"},
		{"10.0.0.5 - 10.0.0.5", "10.0.0.5", "10.0.0.5"},
		{"10.0.0.64/26", "10.0.0.64", "10.0.0.127"},
		{"10.0.0.64 / 26", "10.0.0.64", "10.0.0.127"},
		{"10.0.0.5/32", "10.0.0.5", "10.0.0.5"},
		{"0.0.0.0/0", "0.0.0.0", "255.255.255.255"},
	}
	for _, tt := range tests {
		first, last, err := ParsePool(tt.in)
		if err != nil {
			t.Errorf("ParsePool(%q): %v", tt.in, err)
			continue
		}
		if first != netip.MustParseAddr(tt.first) || last != netip.MustParseAddr(tt.last) {
			t.Errorf("ParsePool(%q) = %s - %s, want %s - %s", tt.in, first, last, tt.first, tt.last)
		}
	}
}

func TestParsePoolInvalid(t *testing.T) {
	for _, in := range []string{
		"",
		"10.0.0.10",
		"10.0.0.20 - 10.0.0.10",
		"10.0.0.300 - 10.0.0.310",
		"10.0.0.5/28",
		"10.0.0.0/33",
		"2001:db8::1 - 2001:db8::2",
		"2001:db8::/64",
		"10.0.0.1 - 2001:db8::1",
	} {
		if _, _, err := ParsePool(in); err == nil {
			t.Errorf("ParsePool(%q): expected error", in)
		}
	}
}
