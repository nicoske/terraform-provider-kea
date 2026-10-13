package dhcp4

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ datasource.DataSourceWithConfigure = (*ConfigHashDataSource)(nil)

type ConfigHashDataSource struct {
	client *keadhcp4.Client
}

func (d *ConfigHashDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_config_hash"
}

func (d *ConfigHashDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the hash of the running configuration of the Kea DHCPv4 server, " +
			"to detect changes made outside Terraform.",
		Attributes: map[string]schema.Attribute{
			"hash": schema.StringAttribute{
				Description: "SHA-256 hash of the running configuration (`config-hash-get`).",
				Computed:    true,
			},
		},
	}
}

func (d *ConfigHashDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	keaClients, ok := req.ProviderData.(clients.KeaClients)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	if keaClients.DHCP4 == nil {
		resp.Diagnostics.Append(errors.NewUnconfiguredClientError("dhcp4"))
		return
	}

	d.client = keaClients.DHCP4
}

func (d *ConfigHashDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigHashModel

	hash, err := d.client.ConfigHash(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Configuration Hash", err.Error())
		return
	}

	data.FromAPI(hash)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
