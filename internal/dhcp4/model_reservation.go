package dhcp4

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/keatypes"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

var operationTargets = []kea.OperationTarget{
	kea.OperationTargetDatabase,
	kea.OperationTargetMemory,
	kea.OperationTargetAll,
}

func operationTargetValidator() validator.String {
	values := make([]string, len(operationTargets))
	for i, t := range operationTargets {
		values[i] = string(t)
	}
	return stringvalidator.OneOf(values...)
}

type ReservationModel struct {
	ID             types.String         `tfsdk:"id"`
	SubnetID       types.Int64          `tfsdk:"subnet_id"`
	CircuitID      keatypes.HexID       `tfsdk:"circuit_id"`
	ClientID       keatypes.HexID       `tfsdk:"client_id"`
	DUID           keatypes.HexID       `tfsdk:"duid"`
	FlexID         keatypes.HexID       `tfsdk:"flex_id"`
	HWAddress      hwtypes.MACAddress   `tfsdk:"hw_address"`
	IPAddress      iptypes.IPv4Address  `tfsdk:"ip_address"`
	BootFileName   types.String         `tfsdk:"boot_file_name"`
	ClientClasses  types.Set            `tfsdk:"client_classes"`
	Hostname       types.String         `tfsdk:"hostname"`
	NextServer     iptypes.IPv4Address  `tfsdk:"next_server"`
	ServerHostname types.String         `tfsdk:"server_hostname"`
	OptionData     types.Set            `tfsdk:"option_data"`
	UserContext    jsontypes.Normalized `tfsdk:"user_context"`

	OperationTarget types.String `tfsdk:"operation_target"`
}

func (m *ReservationModel) ToAPI(ctx context.Context) (keadhcp4.Reservation, diag.Diagnostics) {
	var diags diag.Diagnostics

	reservation := keadhcp4.Reservation{
		SubnetID:       uint32(m.SubnetID.ValueInt64()),
		CircuitID:      hexID(m.CircuitID),
		ClientID:       hexID(m.ClientID),
		DUID:           hexID(m.DUID),
		FlexID:         hexID(m.FlexID),
		HWAddress:      macAddrToHexID(m.HWAddress),
		IPAddress:      ipv4Pointer(m.IPAddress),
		BootFileName:   m.BootFileName.ValueStringPointer(),
		Hostname:       m.Hostname.ValueStringPointer(),
		NextServer:     ipv4Pointer(m.NextServer),
		ServerHostname: m.ServerHostname.ValueStringPointer(),
	}

	if !m.ClientClasses.IsNull() && !m.ClientClasses.IsUnknown() {
		diags.Append(m.ClientClasses.ElementsAs(ctx, &reservation.ClientClasses, false)...)
	}

	optionData, d := optionDataToAPI(ctx, m.OptionData)
	diags.Append(d...)
	reservation.OptionData = optionData

	if !m.UserContext.IsNull() && !m.UserContext.IsUnknown() {
		userContext := json.RawMessage(m.UserContext.ValueString())
		reservation.UserContext = &userContext
	}

	return reservation, diags
}

func (m *ReservationModel) FromAPI(ctx context.Context, r *keadhcp4.Reservation) diag.Diagnostics {
	var diags diag.Diagnostics

	m.SubnetID = types.Int64Value(int64(r.SubnetID))
	m.CircuitID = keatypes.NewHexIDPointerValue(hexIDToString(r.CircuitID))
	m.ClientID = keatypes.NewHexIDPointerValue(hexIDToString(r.ClientID))
	m.DUID = keatypes.NewHexIDPointerValue(hexIDToString(r.DUID))
	m.FlexID = keatypes.NewHexIDPointerValue(hexIDToString(r.FlexID))
	m.HWAddress = hwtypes.NewMACAddressPointerValue(hexIDToString(r.HWAddress))
	m.IPAddress = iptypes.NewIPv4AddressPointerValue(addrPointerToString(r.IPAddress))
	m.BootFileName = types.StringPointerValue(r.BootFileName)
	m.Hostname = types.StringPointerValue(r.Hostname)
	m.NextServer = iptypes.NewIPv4AddressPointerValue(addrPointerToString(r.NextServer))
	m.ServerHostname = types.StringPointerValue(r.ServerHostname)

	if len(r.ClientClasses) > 0 {
		setVal, d := types.SetValueFrom(ctx, types.StringType, r.ClientClasses)
		diags.Append(d...)
		m.ClientClasses = setVal
	} else {
		m.ClientClasses = types.SetNull(types.StringType)
	}

	optionData, d := optionDataFromAPI(ctx, m.OptionData, r.OptionData)
	diags.Append(d...)
	m.OptionData = optionData

	if r.UserContext != nil {
		m.UserContext = jsontypes.NewNormalizedValue(string(*r.UserContext))
	} else {
		m.UserContext = jsontypes.NewNormalizedNull()
	}

	m.ID = types.StringValue(m.ComputeID())

	return diags
}

func (m *ReservationModel) Target(fallback kea.OperationTarget) kea.OperationTarget {
	if m.OperationTarget.IsNull() || m.OperationTarget.IsUnknown() {
		return fallback
	}
	return kea.OperationTarget(m.OperationTarget.ValueString())
}

func (m *ReservationModel) BuildQuery() keaquery.ReservationQuery {
	subnetID := uint32(m.SubnetID.ValueInt64())

	identifierType, identifier := m.getIdentifier()
	if identifierType != "" {
		return keaquery.ReservationByIdentifier(subnetID, identifierType, identifier)
	}

	ip, _ := m.IPAddress.ValueIPv4Address()
	return keaquery.ReservationByIP(subnetID, ip)
}

func (m *ReservationModel) ComputeID() string {
	subnetID := m.SubnetID.ValueInt64()

	identifierType, identifier := m.getIdentifier()
	if identifierType != "" {
		return fmt.Sprintf("%d/%s/%s", subnetID, identifierType, identifier)
	}

	return fmt.Sprintf("%d/ip-address/%s", subnetID, m.IPAddress.ValueString())
}

// parseReservationID parses an ID in the format produced by ComputeID.
func parseReservationID(id string) (subnetID int64, identifierType, identifier string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[2] == "" {
		return 0, "", "", fmt.Errorf("expected {subnet_id}/{identifier_type}/{identifier}, got %q", id)
	}

	subnetID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || subnetID < 0 || subnetID > math.MaxUint32 {
		return 0, "", "", fmt.Errorf("subnet_id must be an integer between 0 and %d, got %q", uint32(math.MaxUint32), parts[0])
	}

	identifierType, identifier = parts[1], parts[2]
	switch identifierType {
	case "circuit-id", "client-id", "duid", "flex-id":
		_, err = kea.ParseHexID(identifier)
	case "hw-address":
		_, err = net.ParseMAC(identifier)
	case "ip-address":
		var ip netip.Addr
		ip, err = netip.ParseAddr(identifier)
		if err == nil && !ip.Is4() {
			err = fmt.Errorf("not an IPv4 address")
		}
	default:
		return 0, "", "", fmt.Errorf("unsupported identifier type %q, expected one of circuit-id, client-id, duid, flex-id, hw-address or ip-address", identifierType)
	}
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid %s %q: %w", identifierType, identifier, err)
	}

	return subnetID, identifierType, identifier, nil
}

func (m *ReservationModel) getIdentifier() (identifierType, identifier string) {
	switch {
	case !m.CircuitID.IsNull() && !m.CircuitID.IsUnknown():
		return "circuit-id", m.CircuitID.ValueString()
	case !m.ClientID.IsNull() && !m.ClientID.IsUnknown():
		return "client-id", m.ClientID.ValueString()
	case !m.DUID.IsNull() && !m.DUID.IsUnknown():
		return "duid", m.DUID.ValueString()
	case !m.FlexID.IsNull() && !m.FlexID.IsUnknown():
		return "flex-id", m.FlexID.ValueString()
	case !m.HWAddress.IsNull() && !m.HWAddress.IsUnknown():
		return "hw-address", m.HWAddress.ValueString()
	default:
		return "", ""
	}
}

func hexIDToString(id kea.HexID) *string {
	if len(id) == 0 {
		return nil
	}
	s := id.String()
	return &s
}

func addrPointerToString(ip *netip.Addr) *string {
	if ip == nil {
		return nil
	}
	s := ip.String()
	return &s
}

func boolPointer(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}

	r := b.ValueBool()
	return &r
}

func hexID(id keatypes.HexID) kea.HexID {
	if id.IsNull() || id.IsUnknown() {
		return nil
	}

	r, _ := id.ValueHexID()
	return r
}

func ipv4Pointer(ip iptypes.IPv4Address) *netip.Addr {
	if ip.IsNull() || ip.IsUnknown() {
		return nil
	}

	r, _ := ip.ValueIPv4Address()
	return &r
}

func stringPointer(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}

	r := s.ValueString()
	return &r
}

func macAddrToHexID(addr hwtypes.MACAddress) kea.HexID {
	if addr.IsNull() || addr.IsUnknown() {
		return nil
	}

	a, _ := addr.ValueMACAddress()
	id, _ := kea.ParseHexID(a.String())
	return id
}
