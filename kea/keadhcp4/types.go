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

	CircuitID kea.HexID `json:"circuit-id,omitempty"`
	ClientID  kea.HexID `json:"client-id,omitempty"`
	DUID      kea.HexID `json:"duid,omitempty"`
	FlexID    kea.HexID `json:"flex-id,omitempty"`
	HWAddress kea.HexID `json:"hw-address,omitempty"`

	BootFileName   *string          `json:"boot-file-name,omitempty"`
	ClientClasses  []string         `json:"client-classes,omitempty"`
	Hostname       *string          `json:"hostname,omitempty"`
	IPAddress      *netip.Addr      `json:"ip-address,omitempty"`
	NextServer     *netip.Addr      `json:"next-server,omitempty"`
	OptionData     []OptionData     `json:"option-data,omitempty"`
	ServerHostname *string          `json:"server-hostname,omitempty"`
	UserContext    *json.RawMessage `json:"user-context,omitempty"`
}

type ClientClass struct {
	Name string `json:"name"`

	Test                 *string `json:"test,omitempty"`
	TemplateTest         *string `json:"template-test,omitempty"`
	OnlyInAdditionalList *bool   `json:"only-in-additional-list,omitempty"`

	BootFileName     *string          `json:"boot-file-name,omitempty"`
	NextServer       *netip.Addr      `json:"next-server,omitempty"`
	OptionData       []OptionData     `json:"option-data,omitempty"`
	OptionDef        []OptionDef      `json:"option-def,omitempty"`
	ServerHostname   *string          `json:"server-hostname,omitempty"`
	ValidLifetime    *uint32          `json:"valid-lifetime,omitempty"`
	MinValidLifetime *uint32          `json:"min-valid-lifetime,omitempty"`
	MaxValidLifetime *uint32          `json:"max-valid-lifetime,omitempty"`
	UserContext      *json.RawMessage `json:"user-context,omitempty"`
}

type OptionDef struct {
	Name        string  `json:"name"`
	Code        uint8   `json:"code"`
	Type        string  `json:"type"`
	Array       *bool   `json:"array,omitempty"`
	RecordTypes *string `json:"record-types,omitempty"`
	Space       *string `json:"space,omitempty"`
	Encapsulate *string `json:"encapsulate,omitempty"`
}

type OptionData struct {
	Name          *string           `json:"name,omitempty"`
	Code          *uint8            `json:"code,omitempty"`
	Space         *string           `json:"space,omitempty"`
	Payload       kea.OptionPayload `json:"-"`
	AlwaysSend    *bool             `json:"always-send,omitempty"`
	NeverSend     *bool             `json:"never-send,omitempty"`
	ClientClasses []string          `json:"client-classes,omitempty"`
}

type optionDataWire struct {
	Data      *string `json:"data,omitempty"`
	CSVFormat *bool   `json:"csv-format,omitempty"`
}

func (o OptionData) MarshalJSON() ([]byte, error) {
	type alias OptionData
	var wire optionDataWire
	switch p := o.Payload.(type) {
	case nil:
	case kea.CSVPayload:
		wire.Data, wire.CSVFormat = new(string(p)), new(true)
	case kea.BinaryPayload:
		wire.Data, wire.CSVFormat = new(p.String()), new(false)
	case kea.UnresolvedPayload:
		wire.Data = new(string(p))
	default:
		return nil, fmt.Errorf("unsupported option payload %T", p)
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
		o.Payload = nil
		return nil
	}

	// Kea always sets csv-format in responses, so an absent field means CSV.
	format := kea.CSVFormatOf(aux.CSVFormat)
	if format == kea.CSVFormatUnspecified {
		format = kea.CSVFormatCSV
	}

	p, err := kea.NewOptionPayload(*aux.Data, format)
	if err != nil {
		return err
	}
	o.Payload = p
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

func (c *ClientClass) UnmarshalJSON(data []byte) error {
	type Alias ClientClass
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// The Kea API returns these fields as empty strings when not set,
	// rather than omitting them from the response.
	c.BootFileName = nilIfEmpty(c.BootFileName)
	c.ServerHostname = nilIfEmpty(c.ServerHostname)

	// The Kea API returns "0.0.0.0" for next-server when not set.
	if c.NextServer != nil && (!c.NextServer.IsValid() || c.NextServer.IsUnspecified()) {
		c.NextServer = nil
	}

	return nil
}

func (d *OptionDef) UnmarshalJSON(data []byte) error {
	type Alias OptionDef
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// The Kea API returns these fields as empty strings when not set,
	// rather than omitting them from the response.
	d.RecordTypes = nilIfEmpty(d.RecordTypes)
	d.Encapsulate = nilIfEmpty(d.Encapsulate)

	return nil
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
