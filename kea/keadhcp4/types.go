package keadhcp4

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

type Subnet struct {
	ID     uint32       `json:"id"`
	Subnet netip.Prefix `json:"subnet"`

	SharedNetworkName *string `json:"shared-network-name,omitempty"`

	Pools      []Pool       `json:"pools,omitempty"`
	OptionData []OptionData `json:"option-data,omitempty"`

	ClientClasses             []string `json:"client-classes,omitempty"`
	EvaluateAdditionalClasses []string `json:"evaluate-additional-classes,omitempty"`

	Interface *string `json:"interface,omitempty"`
	Relay     *Relay  `json:"relay,omitempty"`

	NextServer     *netip.Addr `json:"next-server,omitempty"`
	ServerHostname *string     `json:"server-hostname,omitempty"`
	BootFileName   *string     `json:"boot-file-name,omitempty"`

	ValidLifetime    *uint32 `json:"valid-lifetime,omitempty"`
	MinValidLifetime *uint32 `json:"min-valid-lifetime,omitempty"`
	MaxValidLifetime *uint32 `json:"max-valid-lifetime,omitempty"`
	RenewTimer       *uint32 `json:"renew-timer,omitempty"`
	RebindTimer      *uint32 `json:"rebind-timer,omitempty"`

	ReservationsGlobal    *bool `json:"reservations-global,omitempty"`
	ReservationsInSubnet  *bool `json:"reservations-in-subnet,omitempty"`
	ReservationsOutOfPool *bool `json:"reservations-out-of-pool,omitempty"`

	DDNSSendUpdates          *bool   `json:"ddns-send-updates,omitempty"`
	DDNSOverrideNoUpdate     *bool   `json:"ddns-override-no-update,omitempty"`
	DDNSOverrideClientUpdate *bool   `json:"ddns-override-client-update,omitempty"`
	DDNSReplaceClientName    *string `json:"ddns-replace-client-name,omitempty"`
	DDNSGeneratedPrefix      *string `json:"ddns-generated-prefix,omitempty"`
	DDNSQualifyingSuffix     *string `json:"ddns-qualifying-suffix,omitempty"`
	DDNSUpdateOnRenew        *bool   `json:"ddns-update-on-renew,omitempty"`
	DDNSTTL                  *uint32 `json:"ddns-ttl,omitempty"`

	UserContext *json.RawMessage `json:"user-context,omitempty"`
}

type Pool struct {
	Pool string `json:"pool"`

	ClientClasses             []string     `json:"client-classes,omitempty"`
	EvaluateAdditionalClasses []string     `json:"evaluate-additional-classes,omitempty"`
	OptionData                []OptionData `json:"option-data,omitempty"`
}

type Relay struct {
	IPAddresses []netip.Addr `json:"ip-addresses"`
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

func (s *Subnet) UnmarshalJSON(data []byte) error {
	type Alias Subnet
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// The Kea API returns these fields as empty strings when not set,
	// rather than omitting them from the response.
	s.SharedNetworkName = nilIfEmpty(s.SharedNetworkName)
	s.Interface = nilIfEmpty(s.Interface)
	s.ServerHostname = nilIfEmpty(s.ServerHostname)
	s.BootFileName = nilIfEmpty(s.BootFileName)
	s.DDNSReplaceClientName = nilIfEmpty(s.DDNSReplaceClientName)
	s.DDNSGeneratedPrefix = nilIfEmpty(s.DDNSGeneratedPrefix)
	s.DDNSQualifyingSuffix = nilIfEmpty(s.DDNSQualifyingSuffix)

	// Kea treats a next-server of "0.0.0.0" as unset.
	if s.NextServer != nil && (!s.NextServer.IsValid() || s.NextServer.IsUnspecified()) {
		s.NextServer = nil
	}

	// The Kea API always returns relay, with an empty address list when not set.
	if s.Relay != nil && len(s.Relay.IPAddresses) == 0 {
		s.Relay = nil
	}

	return nil
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
