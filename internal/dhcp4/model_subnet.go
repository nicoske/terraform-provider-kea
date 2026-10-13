package dhcp4

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

type SubnetModel struct {
	ID                        types.Int64          `tfsdk:"id"`
	Subnet                    cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	SharedNetworkName         types.String         `tfsdk:"shared_network_name"`
	Pools                     types.List           `tfsdk:"pools"`
	OptionData                types.Set            `tfsdk:"option_data"`
	ClientClasses             types.Set            `tfsdk:"client_classes"`
	EvaluateAdditionalClasses types.Set            `tfsdk:"evaluate_additional_classes"`
	Interface                 types.String         `tfsdk:"interface"`
	Relay                     types.Set            `tfsdk:"relay"`
	NextServer                iptypes.IPv4Address  `tfsdk:"next_server"`
	ServerHostname            types.String         `tfsdk:"server_hostname"`
	BootFileName              types.String         `tfsdk:"boot_file_name"`
	ValidLifetime             types.Int64          `tfsdk:"valid_lifetime"`
	MinValidLifetime          types.Int64          `tfsdk:"min_valid_lifetime"`
	MaxValidLifetime          types.Int64          `tfsdk:"max_valid_lifetime"`
	RenewTimer                types.Int64          `tfsdk:"renew_timer"`
	RebindTimer               types.Int64          `tfsdk:"rebind_timer"`
	ReservationsGlobal        types.Bool           `tfsdk:"reservations_global"`
	ReservationsInSubnet      types.Bool           `tfsdk:"reservations_in_subnet"`
	ReservationsOutOfPool     types.Bool           `tfsdk:"reservations_out_of_pool"`
	DDNSSendUpdates           types.Bool           `tfsdk:"ddns_send_updates"`
	DDNSOverrideNoUpdate      types.Bool           `tfsdk:"ddns_override_no_update"`
	DDNSOverrideClientUpdate  types.Bool           `tfsdk:"ddns_override_client_update"`
	DDNSReplaceClientName     types.String         `tfsdk:"ddns_replace_client_name"`
	DDNSGeneratedPrefix       types.String         `tfsdk:"ddns_generated_prefix"`
	DDNSQualifyingSuffix      types.String         `tfsdk:"ddns_qualifying_suffix"`
	DDNSUpdateOnRenew         types.Bool           `tfsdk:"ddns_update_on_renew"`
	DDNSTTL                   types.Int64          `tfsdk:"ddns_ttl"`
	UserContext               jsontypes.Normalized `tfsdk:"user_context"`
}

type PoolModel struct {
	Pool                      types.String `tfsdk:"pool"`
	ClientClasses             types.Set    `tfsdk:"client_classes"`
	EvaluateAdditionalClasses types.Set    `tfsdk:"evaluate_additional_classes"`
	OptionData                types.Set    `tfsdk:"option_data"`
}

var poolObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"pool":                        types.StringType,
		"client_classes":              types.SetType{ElemType: types.StringType},
		"evaluate_additional_classes": types.SetType{ElemType: types.StringType},
		"option_data":                 types.SetType{ElemType: optionDataObjectType},
	},
}

func (m *SubnetModel) ToAPI(ctx context.Context) (keadhcp4.Subnet, diag.Diagnostics) {
	var diags diag.Diagnostics

	prefix, d := m.Subnet.ValueIPv4Prefix()
	diags.Append(d...)

	subnet := keadhcp4.Subnet{
		ID:                       uint32(m.ID.ValueInt64()),
		Subnet:                   prefix,
		Interface:                stringPointer(m.Interface),
		NextServer:               ipv4Pointer(m.NextServer),
		ServerHostname:           stringPointer(m.ServerHostname),
		BootFileName:             stringPointer(m.BootFileName),
		ValidLifetime:            uint32Pointer(m.ValidLifetime),
		MinValidLifetime:         uint32Pointer(m.MinValidLifetime),
		MaxValidLifetime:         uint32Pointer(m.MaxValidLifetime),
		RenewTimer:               uint32Pointer(m.RenewTimer),
		RebindTimer:              uint32Pointer(m.RebindTimer),
		ReservationsGlobal:       boolPointer(m.ReservationsGlobal),
		ReservationsInSubnet:     boolPointer(m.ReservationsInSubnet),
		ReservationsOutOfPool:    boolPointer(m.ReservationsOutOfPool),
		DDNSSendUpdates:          boolPointer(m.DDNSSendUpdates),
		DDNSOverrideNoUpdate:     boolPointer(m.DDNSOverrideNoUpdate),
		DDNSOverrideClientUpdate: boolPointer(m.DDNSOverrideClientUpdate),
		DDNSReplaceClientName:    stringPointer(m.DDNSReplaceClientName),
		DDNSGeneratedPrefix:      stringPointer(m.DDNSGeneratedPrefix),
		DDNSQualifyingSuffix:     stringPointer(m.DDNSQualifyingSuffix),
		DDNSUpdateOnRenew:        boolPointer(m.DDNSUpdateOnRenew),
		DDNSTTL:                  uint32Pointer(m.DDNSTTL),
	}

	diags.Append(stringSetToAPI(ctx, m.ClientClasses, &subnet.ClientClasses)...)
	diags.Append(stringSetToAPI(ctx, m.EvaluateAdditionalClasses, &subnet.EvaluateAdditionalClasses)...)

	if !m.Relay.IsNull() && !m.Relay.IsUnknown() {
		var addresses []iptypes.IPv4Address
		diags.Append(m.Relay.ElementsAs(ctx, &addresses, false)...)
		relay := keadhcp4.Relay{}
		for _, a := range addresses {
			if ip := ipv4Pointer(a); ip != nil {
				relay.IPAddresses = append(relay.IPAddresses, *ip)
			}
		}
		subnet.Relay = &relay
	}

	if !m.Pools.IsNull() && !m.Pools.IsUnknown() {
		var pools []PoolModel
		diags.Append(m.Pools.ElementsAs(ctx, &pools, false)...)
		for _, p := range pools {
			pool, d := p.ToAPI(ctx)
			diags.Append(d...)
			subnet.Pools = append(subnet.Pools, pool)
		}
	}

	optionData, d := optionDataToAPI(ctx, m.OptionData)
	diags.Append(d...)
	subnet.OptionData = optionData

	if !m.UserContext.IsNull() && !m.UserContext.IsUnknown() {
		userContext := json.RawMessage(m.UserContext.ValueString())
		subnet.UserContext = &userContext
	}

	return subnet, diags
}

func (m *SubnetModel) FromAPI(ctx context.Context, s *keadhcp4.Subnet) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.Int64Value(int64(s.ID))
	m.Subnet = cidrtypes.NewIPv4PrefixValue(s.Subnet.String())
	m.SharedNetworkName = types.StringPointerValue(s.SharedNetworkName)
	m.Interface = types.StringPointerValue(s.Interface)
	m.NextServer = iptypes.NewIPv4AddressPointerValue(addrPointerToString(s.NextServer))
	m.ServerHostname = types.StringPointerValue(s.ServerHostname)
	m.BootFileName = types.StringPointerValue(s.BootFileName)
	m.ValidLifetime = int64PointerValue(s.ValidLifetime)
	m.MinValidLifetime = int64PointerValue(s.MinValidLifetime)
	m.MaxValidLifetime = int64PointerValue(s.MaxValidLifetime)
	m.RenewTimer = int64PointerValue(s.RenewTimer)
	m.RebindTimer = int64PointerValue(s.RebindTimer)
	m.ReservationsGlobal = types.BoolPointerValue(s.ReservationsGlobal)
	m.ReservationsInSubnet = types.BoolPointerValue(s.ReservationsInSubnet)
	m.ReservationsOutOfPool = types.BoolPointerValue(s.ReservationsOutOfPool)
	m.DDNSSendUpdates = types.BoolPointerValue(s.DDNSSendUpdates)
	m.DDNSOverrideNoUpdate = types.BoolPointerValue(s.DDNSOverrideNoUpdate)
	m.DDNSOverrideClientUpdate = types.BoolPointerValue(s.DDNSOverrideClientUpdate)
	m.DDNSReplaceClientName = types.StringPointerValue(s.DDNSReplaceClientName)
	m.DDNSGeneratedPrefix = types.StringPointerValue(s.DDNSGeneratedPrefix)
	m.DDNSQualifyingSuffix = types.StringPointerValue(s.DDNSQualifyingSuffix)
	m.DDNSUpdateOnRenew = types.BoolPointerValue(s.DDNSUpdateOnRenew)
	m.DDNSTTL = int64PointerValue(s.DDNSTTL)

	var d diag.Diagnostics
	m.ClientClasses, d = stringSetFromAPI(ctx, s.ClientClasses)
	diags.Append(d...)
	m.EvaluateAdditionalClasses, d = stringSetFromAPI(ctx, s.EvaluateAdditionalClasses)
	diags.Append(d...)

	if s.Relay != nil && len(s.Relay.IPAddresses) > 0 {
		var addresses []attr.Value
		for _, a := range s.Relay.IPAddresses {
			addresses = append(addresses, iptypes.NewIPv4AddressValue(a.String()))
		}
		m.Relay, d = types.SetValue(iptypes.IPv4AddressType{}, addresses)
		diags.Append(d...)
	} else {
		m.Relay = types.SetNull(iptypes.IPv4AddressType{})
	}

	m.Pools, d = poolsFromAPI(ctx, m.Pools, s.Pools)
	diags.Append(d...)

	m.OptionData, d = optionDataFromAPI(ctx, m.OptionData, s.OptionData)
	diags.Append(d...)

	if s.UserContext != nil {
		// Kea spaces out JSON, jsonencode does not.
		var compact bytes.Buffer
		if err := json.Compact(&compact, *s.UserContext); err != nil {
			diags.AddError("Invalid User Context", err.Error())
		}
		m.UserContext = jsontypes.NewNormalizedValue(compact.String())
	} else {
		m.UserContext = jsontypes.NewNormalizedNull()
	}

	return diags
}

func (m *SubnetModel) BuildQuery() keaquery.SubnetQuery {
	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		return keaquery.SubnetByID(uint32(m.ID.ValueInt64()))
	}

	prefix, _ := m.Subnet.ValueIPv4Prefix()
	return keaquery.SubnetByPrefix(prefix)
}

func (p *PoolModel) ToAPI(ctx context.Context) (keadhcp4.Pool, diag.Diagnostics) {
	var diags diag.Diagnostics

	pool := keadhcp4.Pool{
		Pool: p.Pool.ValueString(),
	}

	diags.Append(stringSetToAPI(ctx, p.ClientClasses, &pool.ClientClasses)...)
	diags.Append(stringSetToAPI(ctx, p.EvaluateAdditionalClasses, &pool.EvaluateAdditionalClasses)...)

	optionData, d := optionDataToAPI(ctx, p.OptionData)
	diags.Append(d...)
	pool.OptionData = optionData

	return pool, diags
}

// poolsFromAPI keeps the configured spelling and order of each pool whose
// range Kea reports, since Kea sorts pools and rewrites ranges ("a - b" to
// "a-b", or to a prefix when the range is one).
func poolsFromAPI(ctx context.Context, prior types.List, pools []keadhcp4.Pool) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(pools) == 0 {
		return types.ListNull(poolObjectType), diags
	}

	var priorModels []PoolModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
	}

	priorIndex := make([]int, len(pools))
	for i, pool := range pools {
		priorIndex[i] = -1
		first, last, err := keadhcp4.ParsePool(pool.Pool)
		if err != nil {
			continue
		}
		for j, pm := range priorModels {
			pf, pl, err := keadhcp4.ParsePool(pm.Pool.ValueString())
			if err == nil && pf == first && pl == last {
				priorIndex[i] = j
				break
			}
		}
	}

	order := make([]int, len(pools))
	for i := range order {
		order[i] = i
	}
	rank := func(i int) int {
		if priorIndex[i] < 0 {
			return len(priorModels)
		}
		return priorIndex[i]
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(rank(a), rank(b))
	})

	var models []PoolModel
	for _, i := range order {
		pool := pools[i]
		model := PoolModel{
			Pool:       types.StringValue(pool.Pool),
			OptionData: types.SetNull(optionDataObjectType),
		}
		if j := priorIndex[i]; j >= 0 {
			model.Pool = priorModels[j].Pool
			model.OptionData = priorModels[j].OptionData
		}

		var d diag.Diagnostics
		model.ClientClasses, d = stringSetFromAPI(ctx, pool.ClientClasses)
		diags.Append(d...)
		model.EvaluateAdditionalClasses, d = stringSetFromAPI(ctx, pool.EvaluateAdditionalClasses)
		diags.Append(d...)
		model.OptionData, d = optionDataFromAPI(ctx, model.OptionData, pool.OptionData)
		diags.Append(d...)

		models = append(models, model)
	}

	list, d := types.ListValueFrom(ctx, poolObjectType, models)
	diags.Append(d...)
	return list, diags
}

func stringSetToAPI(ctx context.Context, set types.Set, target *[]string) diag.Diagnostics {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	return set.ElementsAs(ctx, target, false)
}

func stringSetFromAPI(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if len(values) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, values)
}
