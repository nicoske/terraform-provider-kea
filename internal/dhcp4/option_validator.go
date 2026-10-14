package dhcp4

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type optionValueValidator struct{}

func (optionValueValidator) Description(context.Context) string {
	return "data must be valid for csv_format"
}

func (v optionValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (optionValueValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()
	data, _ := attrs["data"].(types.String)
	csvFormat, _ := attrs["csv_format"].(types.Bool)
	if _, err := parseOptionValue(data, csvFormat); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path.AtName("data"), "Invalid Option Data", err.Error())
	}
}
