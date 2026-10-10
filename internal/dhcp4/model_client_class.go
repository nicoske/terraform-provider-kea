package dhcp4

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

type ClientClassModel struct {
	ID                   types.String         `tfsdk:"id"`
	Name                 types.String         `tfsdk:"name"`
	Test                 types.String         `tfsdk:"test"`
	TemplateTest         types.String         `tfsdk:"template_test"`
	OnlyInAdditionalList types.Bool           `tfsdk:"only_in_additional_list"`
	BootFileName         types.String         `tfsdk:"boot_file_name"`
	NextServer           iptypes.IPv4Address  `tfsdk:"next_server"`
	ServerHostname       types.String         `tfsdk:"server_hostname"`
	ValidLifetime        types.Int64          `tfsdk:"valid_lifetime"`
	MinValidLifetime     types.Int64          `tfsdk:"min_valid_lifetime"`
	MaxValidLifetime     types.Int64          `tfsdk:"max_valid_lifetime"`
	OptionData           types.Set            `tfsdk:"option_data"`
	OptionDef            types.List           `tfsdk:"option_def"`
	UserContext          jsontypes.Normalized `tfsdk:"user_context"`
}

type OptionDefModel struct {
	Name        types.String `tfsdk:"name"`
	Code        types.Int32  `tfsdk:"code"`
	Type        types.String `tfsdk:"type"`
	Array       types.Bool   `tfsdk:"array"`
	RecordTypes types.String `tfsdk:"record_types"`
	Space       types.String `tfsdk:"space"`
	Encapsulate types.String `tfsdk:"encapsulate"`
}

// ClientClassResourceModel adds the attributes that only the resource has.
type ClientClassResourceModel struct {
	ClientClassModel
	InSync types.Bool `tfsdk:"in_sync"`
}

func (m *ClientClassModel) ToAPI(ctx context.Context) (keadhcp4.ClientClass, diag.Diagnostics) {
	var diags diag.Diagnostics

	class := keadhcp4.ClientClass{
		Name:             m.Name.ValueString(),
		Test:             stringPointer(m.Test),
		TemplateTest:     stringPointer(m.TemplateTest),
		BootFileName:     stringPointer(m.BootFileName),
		NextServer:       ipv4Pointer(m.NextServer),
		ServerHostname:   stringPointer(m.ServerHostname),
		ValidLifetime:    uint32Pointer(m.ValidLifetime),
		MinValidLifetime: uint32Pointer(m.MinValidLifetime),
		MaxValidLifetime: uint32Pointer(m.MaxValidLifetime),
	}

	if m.OnlyInAdditionalList.ValueBool() {
		class.OnlyInAdditionalList = boolPointer(m.OnlyInAdditionalList)
	}

	optionData, d := optionDataToAPI(ctx, m.OptionData)
	diags.Append(d...)
	class.OptionData = optionData

	if !m.OptionDef.IsNull() && !m.OptionDef.IsUnknown() {
		var optionDefModels []OptionDefModel
		diags.Append(m.OptionDef.ElementsAs(ctx, &optionDefModels, false)...)

		for _, od := range optionDefModels {
			class.OptionDef = append(class.OptionDef, od.ToAPI())
		}
	}

	if !m.UserContext.IsNull() && !m.UserContext.IsUnknown() {
		userContext := json.RawMessage(m.UserContext.ValueString())
		class.UserContext = &userContext
	}

	return class, diags
}

func (m *ClientClassModel) FromAPI(ctx context.Context, c *keadhcp4.ClientClass) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(c.Name)
	m.Name = types.StringValue(c.Name)
	m.Test = types.StringPointerValue(c.Test)
	m.TemplateTest = types.StringPointerValue(c.TemplateTest)
	m.OnlyInAdditionalList = types.BoolValue(c.OnlyInAdditionalList != nil && *c.OnlyInAdditionalList)
	m.BootFileName = types.StringPointerValue(c.BootFileName)
	m.NextServer = iptypes.NewIPv4AddressPointerValue(addrPointerToString(c.NextServer))
	m.ServerHostname = types.StringPointerValue(c.ServerHostname)
	m.ValidLifetime = int64PointerValue(c.ValidLifetime)
	m.MinValidLifetime = int64PointerValue(c.MinValidLifetime)
	m.MaxValidLifetime = int64PointerValue(c.MaxValidLifetime)

	optionData, d := optionDataFromAPI(ctx, m.OptionData, c.OptionData)
	diags.Append(d...)
	m.OptionData = optionData

	if len(c.OptionDef) > 0 {
		optionDefModels := make([]OptionDefModel, len(c.OptionDef))
		for i, od := range c.OptionDef {
			optionDefModels[i].FromAPI(&od)
		}
		optionDefList, d := types.ListValueFrom(ctx, m.OptionDef.ElementType(ctx), optionDefModels)
		diags.Append(d...)
		m.OptionDef = optionDefList
	} else {
		m.OptionDef = types.ListNull(m.OptionDef.ElementType(ctx))
	}

	if c.UserContext != nil {
		m.UserContext = jsontypes.NewNormalizedValue(string(*c.UserContext))
	} else {
		m.UserContext = jsontypes.NewNormalizedNull()
	}

	return diags
}

func (o *OptionDefModel) ToAPI() keadhcp4.OptionDef {
	return keadhcp4.OptionDef{
		Name:        o.Name.ValueString(),
		Code:        uint8(o.Code.ValueInt32()),
		Type:        o.Type.ValueString(),
		Array:       boolPointer(o.Array),
		RecordTypes: stringPointer(o.RecordTypes),
		Space:       stringPointer(o.Space),
		Encapsulate: stringPointer(o.Encapsulate),
	}
}

func (o *OptionDefModel) FromAPI(od *keadhcp4.OptionDef) {
	o.Name = types.StringValue(od.Name)
	o.Code = types.Int32Value(int32(od.Code))
	o.Type = types.StringValue(od.Type)
	o.Array = types.BoolValue(od.Array != nil && *od.Array)
	o.RecordTypes = types.StringPointerValue(od.RecordTypes)
	o.Space = types.StringPointerValue(od.Space)
	o.Encapsulate = types.StringPointerValue(od.Encapsulate)
}

func uint32Pointer(i types.Int64) *uint32 {
	if i.IsNull() || i.IsUnknown() {
		return nil
	}

	r := uint32(i.ValueInt64())
	return &r
}

func int64PointerValue(i *uint32) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}
