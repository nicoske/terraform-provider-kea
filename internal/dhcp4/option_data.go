package dhcp4

import (
	"context"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

type OptionDataModel struct {
	Name          types.String `tfsdk:"name"`
	Code          types.Int32  `tfsdk:"code"`
	Space         types.String `tfsdk:"space"`
	Data          types.String `tfsdk:"data"`
	CSVFormat     types.Bool   `tfsdk:"csv_format"`
	AlwaysSend    types.Bool   `tfsdk:"always_send"`
	NeverSend     types.Bool   `tfsdk:"never_send"`
	ClientClasses types.Set    `tfsdk:"client_classes"`
}

var optionDataObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"code":           types.Int32Type,
		"space":          types.StringType,
		"data":           types.StringType,
		"csv_format":     types.BoolType,
		"always_send":    types.BoolType,
		"never_send":     types.BoolType,
		"client_classes": types.SetType{ElemType: types.StringType},
	},
}

func optionDataBlock(description string) schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description: "Option name (e.g., `domain-name-servers`). Required if `code` is not set.",
					Optional:    true,
					Computed:    true,
				},
				"code": schema.Int32Attribute{
					Description: "Option code (0-255). Required if `name` is not set.",
					Optional:    true,
					Computed:    true,
					Validators: []validator.Int32{
						int32validator.Between(0, math.MaxUint8),
					},
				},
				"space": schema.StringAttribute{
					Description: "Option space. Defaults to `dhcp4`.",
					Optional:    true,
					Computed:    true,
				},
				"data": schema.StringAttribute{
					Description: "Option value. Format depends on `csv_format`.",
					Optional:    true,
				},
				"csv_format": schema.BoolAttribute{
					Description: "When `true` (default), data is comma-separated values. " +
						"When `false`, data is raw bytes as hex (`dead` or `0xdead`), colon/space-delimited octets (`de:ad` or `de ad`), or a quoted string (`'text'`).",
					Optional: true,
					Computed: true,
				},
				"always_send": schema.BoolAttribute{
					Description: "Always send this option, even if not requested by the client.",
					Optional:    true,
					Computed:    true,
				},
				"never_send": schema.BoolAttribute{
					Description: "Never send this option to the client.",
					Optional:    true,
					Computed:    true,
				},
				"client_classes": schema.SetAttribute{
					Description: "Only send this option to clients in these classes. " +
						"Requires Kea 3.1.0 or newer.",
					Optional:    true,
					ElementType: types.StringType,
				},
			},
			Validators: []validator.Object{
				optionValueValidator{},
				objectvalidator.AtLeastOneOf(
					path.MatchRoot("name"),
					path.MatchRoot("code"),
				),
			},
		},
	}
}

func optionDataDataSourceAttribute(description string) dsschema.SetNestedAttribute {
	return dsschema.SetNestedAttribute{
		Description: description,
		Computed:    true,
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				"name": dsschema.StringAttribute{
					Description: "Option name.",
					Computed:    true,
				},
				"code": dsschema.Int32Attribute{
					Description: "Option code.",
					Computed:    true,
				},
				"space": dsschema.StringAttribute{
					Description: "Option space.",
					Computed:    true,
				},
				"data": dsschema.StringAttribute{
					Description: "Option value.",
					Computed:    true,
				},
				"csv_format": dsschema.BoolAttribute{
					Description: "Whether data is in CSV format (`true`) or raw format (`false`).",
					Computed:    true,
				},
				"always_send": dsschema.BoolAttribute{
					Description: "Whether this option is always sent to the client.",
					Computed:    true,
				},
				"never_send": dsschema.BoolAttribute{
					Description: "Whether this option is never sent to the client.",
					Computed:    true,
				},
				"client_classes": dsschema.SetAttribute{
					Description: "Client classes for which this option applies.",
					Computed:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}

func optionDataToAPI(ctx context.Context, set types.Set) ([]keadhcp4.OptionData, diag.Diagnostics) {
	var diags diag.Diagnostics

	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	var models []OptionDataModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)

	var options []keadhcp4.OptionData
	for _, od := range models {
		option, d := od.ToAPI(ctx)
		diags.Append(d...)
		options = append(options, option)
	}

	return options, diags
}

// optionDataFromAPI keeps the configured spelling of data for every option
// that the prior value describes, so equivalent spellings do not cause drift.
func optionDataFromAPI(ctx context.Context, prior types.Set, options []keadhcp4.OptionData) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(options) == 0 {
		return types.SetNull(prior.ElementType(ctx)), diags
	}

	var priorModels []OptionDataModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
	}
	modelIndex, d := matchOptions(ctx, priorModels, options)
	diags.Append(d...)

	var models []OptionDataModel
	for i, od := range options {
		var model OptionDataModel
		diags.Append(model.FromAPI(ctx, &od)...)
		if j := modelIndex[i]; j >= 0 {
			model.Data = priorModels[j].Data
		}
		models = append(models, model)
	}

	set, d := types.SetValueFrom(ctx, prior.ElementType(ctx), models)
	diags.Append(d...)
	return set, diags
}

func (o *OptionDataModel) ToAPI(ctx context.Context) (keadhcp4.OptionData, diag.Diagnostics) {
	var diags diag.Diagnostics

	option := keadhcp4.OptionData{
		Name:       stringPointer(o.Name),
		Space:      stringPointer(o.Space),
		AlwaysSend: boolPointer(o.AlwaysSend),
		NeverSend:  boolPointer(o.NeverSend),
	}

	if !o.Code.IsNull() && !o.Code.IsUnknown() {
		code := uint8(o.Code.ValueInt32())
		option.Code = &code
	}

	value, err := parseOptionValue(o.Data, o.CSVFormat)
	if err != nil {
		diags.AddError("Invalid Option Data", err.Error())
	}
	option.Value = value

	if !o.ClientClasses.IsNull() && !o.ClientClasses.IsUnknown() {
		diags.Append(o.ClientClasses.ElementsAs(ctx, &option.ClientClasses, false)...)
	}

	return option, diags
}

func (o *OptionDataModel) FromAPI(ctx context.Context, od *keadhcp4.OptionData) diag.Diagnostics {
	var diags diag.Diagnostics

	o.Name = types.StringPointerValue(od.Name)
	if od.Code != nil {
		o.Code = types.Int32Value(int32(*od.Code))
	} else {
		o.Code = types.Int32Null()
	}
	o.Space = types.StringPointerValue(od.Space)
	o.Data, o.CSVFormat = types.StringNull(), types.BoolNull()
	switch p := od.Value.(type) {
	case kea.CSVString:
		o.Data, o.CSVFormat = types.StringValue(string(p)), types.BoolValue(true)
	case kea.ByteString:
		o.Data, o.CSVFormat = types.StringValue(p.String()), types.BoolValue(false)
	}
	o.AlwaysSend = types.BoolPointerValue(od.AlwaysSend)
	o.NeverSend = types.BoolPointerValue(od.NeverSend)

	if len(od.ClientClasses) > 0 {
		setVal, d := types.SetValueFrom(ctx, types.StringType, od.ClientClasses)
		diags.Append(d...)
		o.ClientClasses = setVal
	} else {
		o.ClientClasses = types.SetNull(types.StringType)
	}

	return diags
}

func parseOptionValue(data types.String, csvFormat types.Bool) (kea.OptionValue, error) {
	if data.IsNull() || data.IsUnknown() {
		return nil, nil
	}
	return kea.NewOptionValue(data.ValueString(), kea.CSVFormatOf(boolPointer(csvFormat)))
}
