package dhcp4

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		Description: "Retrieves the hash of the running configuration of every Kea DHCPv4 server of the provider, " +
			"to detect changes made outside Terraform or servers that are out of sync.\n\n" +
			"Servers with different configuration files never have the same hash, " +
			"for example the peers of a high availability pair with their own `this-server-name`.",
		Attributes: map[string]schema.Attribute{
			"hashes": schema.MapAttribute{
				Description: "SHA-256 hash of the running configuration (`config-hash-get`), by server address.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"in_sync": schema.BoolAttribute{
				Description: "Whether all servers have the same hash.",
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

	hashes, err := d.client.ConfigHashes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Configuration Hash", err.Error())
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, hashes)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
