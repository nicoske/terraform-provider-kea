package keadhcp4

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

type Subnet struct {
	ID     uint32 `json:"id"`
	Subnet string `json:"subnet"`

	SharedNetworkName string `json:"shared-network-name,omitempty"`
}

type Reservation struct {
	SubnetID uint32 `json:"subnet-id"`

	CircuitID kea.ByteString `json:"circuit-id,omitempty"`
	ClientID  kea.ByteString `json:"client-id,omitempty"`
	DUID      kea.ByteString `json:"duid,omitempty"`
	FlexID    kea.ByteString `json:"flex-id,omitempty"`
	HWAddress kea.ByteString `json:"hw-address,omitempty"`

	BootFileName   *string          `json:"boot-file-name,omitempty"`
	ClientClasses  []string         `json:"client-classes,omitempty"`
	Hostname       *string          `json:"hostname,omitempty"`
	IPAddress      *netip.Addr      `json:"ip-address,omitempty"`
	NextServer     *netip.Addr      `json:"next-server,omitempty"`
	OptionData     []OptionData     `json:"option-data,omitempty"`
	ServerHostname *string          `json:"server-hostname,omitempty"`
	UserContext    *json.RawMessage `json:"user-context,omitempty"`
}

type OptionData struct {
	Name          *string         `json:"name,omitempty"`
	Code          *uint8          `json:"code,omitempty"`
	Space         *string         `json:"space,omitempty"`
	Value         kea.OptionValue `json:"-"`
	AlwaysSend    *bool           `json:"always-send,omitempty"`
	NeverSend     *bool           `json:"never-send,omitempty"`
	ClientClasses []string        `json:"client-classes,omitempty"`
}

type optionDataWire struct {
	Data      *string `json:"data,omitempty"`
	CSVFormat *bool   `json:"csv-format,omitempty"`
}

func (o OptionData) MarshalJSON() ([]byte, error) {
	type alias OptionData
	var wire optionDataWire
	switch p := o.Value.(type) {
	case nil:
	case kea.CSVString:
		wire.Data, wire.CSVFormat = new(string(p)), new(true)
	case kea.ByteString:
		wire.Data, wire.CSVFormat = new(p.String()), new(false)
	case kea.UnresolvedString:
		wire.Data = new(string(p))
	default:
		return nil, fmt.Errorf("unsupported option value %T", p)
	}
	return json.Marshal(struct {
		alias
		optionDataWire
	}{alias(o), wire})
}

func (o *OptionData) UnmarshalJSON(b []byte) error {
	type alias OptionData
	aux := struct {
		*alias
		optionDataWire
	}{alias: (*alias)(o)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.Data == nil {
		o.Value = nil
		return nil
	}

	// Kea always sets csv-format in responses, so an absent field means CSV.
	format := kea.CSVFormatOf(aux.CSVFormat)
	if format == kea.CSVFormatUnspecified {
		format = kea.CSVFormatCSV
	}

	p, err := kea.NewOptionValue(*aux.Data, format)
	if err != nil {
		return err
	}
	o.Value = p
	return nil
}

func (r *Reservation) UnmarshalJSON(data []byte) error {
	type Alias Reservation
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// The Kea API returns these fields as empty strings when not set,
	// rather than omitting them from the response.
	r.BootFileName = nilIfEmpty(r.BootFileName)
	r.Hostname = nilIfEmpty(r.Hostname)
	r.ServerHostname = nilIfEmpty(r.ServerHostname)

	// The Kea API returns "0.0.0.0" for next-server when not set.
	if r.NextServer != nil && (!r.NextServer.IsValid() || r.NextServer.IsUnspecified()) {
		r.NextServer = nil
	}

	return nil
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
