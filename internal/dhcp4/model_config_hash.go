package dhcp4

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigHashModel struct {
	Hash types.String `tfsdk:"hash"`
}

func (m *ConfigHashModel) FromAPI(hash string) {
	m.Hash = types.StringValue(hash)
}
