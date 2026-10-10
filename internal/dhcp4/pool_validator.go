package dhcp4

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

type poolValidator struct{}

func (poolValidator) Description(context.Context) string {
	return `pool must be an IPv4 range "first - last" or an IPv4 prefix without host bits`
}

func (v poolValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (poolValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, _, err := keadhcp4.ParsePool(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Pool", err.Error())
	}
}
