package dhcp4

import (
	"context"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigHashModel struct {
	Hashes types.Map  `tfsdk:"hashes"`
	InSync types.Bool `tfsdk:"in_sync"`
}

func (m *ConfigHashModel) FromAPI(ctx context.Context, hashes map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Hashes, diags = types.MapValueFrom(ctx, types.StringType, hashes)
	m.InSync = types.BoolValue(len(slices.Compact(slices.Sorted(maps.Values(hashes)))) <= 1)
	return diags
}
