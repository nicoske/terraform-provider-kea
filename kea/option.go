package kea

import (
	"bytes"
	"fmt"
)

type OptionValue interface {
	isOptionValue()
}

type CSVString string

type UnresolvedString string

func (CSVString) isOptionValue()        {}
func (ByteString) isOptionValue()       {}
func (UnresolvedString) isOptionValue() {}

// CSVFormat is the tri-state of Kea's csv-format field. Unspecified means the
// field was omitted and Kea decides from the option definition.
type CSVFormat int

const (
	CSVFormatUnspecified CSVFormat = iota
	CSVFormatCSV
	CSVFormatBinary
)

func CSVFormatOf(v *bool) CSVFormat {
	switch {
	case v == nil:
		return CSVFormatUnspecified
	case *v:
		return CSVFormatCSV
	default:
		return CSVFormatBinary
	}
}

func ValueDescribes(want, have OptionValue) bool {
	switch w := want.(type) {
	case nil:
		return have == nil
	case CSVString:
		h, ok := have.(CSVString)
		return ok && w == h
	case ByteString:
		h, ok := have.(ByteString)
		return ok && bytes.Equal(w, h)
	case UnresolvedString:
		switch h := have.(type) {
		case CSVString:
			return string(w) == string(h)
		case ByteString:
			b, err := ParseByteString(string(w))
			return err == nil && bytes.Equal(b, h)
		}
	}
	return false
}

// NewOptionValue interprets option data the way Kea does for a given
// csv-format. Unspecified means csv-format was omitted: Kea reads the data as
// CSV when the option has a definition and as binary otherwise, so the result
// stays unresolved until Kea reports the format it chose.
func NewOptionValue(data string, format CSVFormat) (OptionValue, error) {
	switch format {
	case CSVFormatUnspecified:
		return UnresolvedString(data), nil
	case CSVFormatCSV:
		return CSVString(data), nil
	case CSVFormatBinary:
		b, err := ParseByteString(data)
		if err != nil {
			return nil, fmt.Errorf("option data %q: %w", data, err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported csv format %d", format)
	}
}
