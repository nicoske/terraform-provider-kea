package kea

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ByteString is a value in Kea's textual binary format: a single-quoted ASCII
// string or a string of hexadecimal digits.
type ByteString []byte

func ParseByteString(s string) (ByteString, error) {
	if b := quotedStringToBinary(s); len(b) > 0 {
		return ByteString(b), nil
	}
	if s == "" {
		return ByteString{}, nil
	}

	b, err := parseHex(s)
	if err != nil {
		return nil, fmt.Errorf("not a quoted string or a string of hexadecimal digits: %w", err)
	}
	return ByteString(b), nil
}

func (k ByteString) String() string {
	return strings.ToUpper(hex.EncodeToString(k))
}

func (k ByteString) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

func (k *ByteString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	id, err := ParseByteString(s)
	if err != nil {
		return err
	}
	*k = id
	return nil
}

func quotedStringToBinary(s string) []byte {
	t := strings.TrimSpace(s)
	if len(t) > 1 && t[0] == '\'' && t[len(t)-1] == '\'' {
		return []byte(strings.TrimSpace(t[1 : len(t)-1]))
	}
	return nil
}

func parseHex(s string) ([]byte, error) {
	if len(s) == 0 {
		return nil, nil
	}
	if strings.Contains(s, ":") {
		return parseHexWithSeparator(s, ":")
	}
	if strings.Contains(s, " ") {
		return parseHexWithSeparator(s, " ")
	}
	if len(s) > 2 && strings.HasPrefix(s, "0x") {
		return parseHexWithPadding(s[2:])
	}
	return parseHexWithPadding(s)
}

func parseHexWithSeparator(s string, sep string) ([]byte, error) {
	parts := strings.Split(s, sep)
	b := make([]byte, len(parts))
	for i, part := range parts {
		if len(part) == 1 {
			part = "0" + part
		} else if len(part) != 2 {
			return nil, fmt.Errorf("invalid segment length: %q", part)
		}

		if _, err := hex.Decode(b[i:i+1], []byte(part)); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func parseHexWithPadding(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}
