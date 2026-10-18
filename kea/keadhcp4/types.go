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
