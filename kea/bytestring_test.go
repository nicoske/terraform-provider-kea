package kea_test

import (
	"bytes"
	"testing"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

func TestParseByteString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"empty string", "", []byte{}, false},

		{"plain even", "aabbcc", []byte{0xaa, 0xbb, 0xcc}, false},
		{"plain odd (padding)", "abc", []byte{0x0a, 0xbc}, false},
		{"uppercase", "AABBCC", []byte{0xaa, 0xbb, 0xcc}, false},

		{"prefix even", "0xaabbcc", []byte{0xaa, 0xbb, 0xcc}, false},
		{"prefix odd (padding)", "0xabc", []byte{0x0a, 0xbc}, false},
		{"bare prefix", "0x", nil, true},

		{"colons", "aa:bb:cc", []byte{0xaa, 0xbb, 0xcc}, false},
		{"colons padding", "a:b:c", []byte{0x0a, 0x0b, 0x0c}, false},
		{"spaces", "aa bb cc", []byte{0xaa, 0xbb, 0xcc}, false},
		{"spaces padding", "a b c", []byte{0x0a, 0x0b, 0x0c}, false},

		{"quoted ascii", "'charter950'", []byte("charter950"), false},
		{"quoted ascii trimmed", "  ' text ' ", []byte("text"), false},
		{"empty quotes", "''", nil, true},

		{"invalid hex chars", "zz:yy", nil, true},
		{"invalid hex chars prefix", "0xzz", nil, true},
		{"invalid prefix", "0XAABBCC", nil, true},

		{"forbidden: prefix + colons", "0xaa:bb:cc", nil, true},
		{"forbidden: prefix + spaces", "0xaa bb cc", nil, true},
		{"forbidden: mixed separators", "aa:bb cc", nil, true},

		{"invalid segment length", "aaa:bbb", nil, true},
		{"trailing separator", "aa:bb:", nil, true},
		{"leading separator", ":aa:bb", nil, true},
		{"double separator", "aa::bb", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kea.ParseByteString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseByteString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !bytes.Equal(got, tt.want) {
					t.Errorf("ParseByteString(%q) = %x, want %x", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestByteStringString(t *testing.T) {
	tests := []struct {
		name string
		val  kea.ByteString
		want string
	}{
		{"nil", nil, ""},
		{"empty", kea.ByteString{}, ""},
		{"single byte", kea.ByteString{0x0a}, "0A"},
		{"multiple bytes", kea.ByteString{0xaa, 0xbb, 0xcc}, "AABBCC"},
		{"leading zero byte", kea.ByteString{0x00, 0x01}, "0001"},
		{"all byte values", kea.ByteString{0x00, 0x0f, 0x10, 0xff}, "000F10FF"},
		{"ascii", kea.ByteString("charter950"), "63686172746572393530"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.val.String(); got != tt.want {
				t.Errorf("ByteString.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
