package dhcp4

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ datasource.DataSourceWithConfigure = (*ClientClassDataSource)(nil)

type ClientClassDataSource struct {
	client *keadhcp4.Client
}

func (d *ClientClassDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_client_class"
}

func (d *ClientClassDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a DHCPv4 client class from the Kea DHCP server.\n\n" +
			"**Important:** Requires the `class_cmds` hook library.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the client class, equal to its name.",
			},
			"name": schema.StringAttribute{
				Description: "Name of the client class.",
				Required:    true,
			},

			"test": schema.StringAttribute{
				Description: "Expression that a client must match to belong to this class.",
				Computed:    true,
			},
			"template_test": schema.StringAttribute{
				Description: "Expression whose result names a subclass the client is assigned to.",
				Computed:    true,
			},
			"only_in_additional_list": schema.BoolAttribute{
				Description: "Whether the class is only evaluated when listed in `evaluate-additional-classes`.",
				Computed:    true,
			},

			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet, Option 67).",
				Computed:    true,
			},
			"next_server": schema.StringAttribute{
				Description: "Next server address for network booting (`siaddr` field in DHCP packet).",
				Computed:    true,
				CustomType:  iptypes.IPv4AddressType{},
			},
			"server_hostname": schema.StringAttribute{
				Description: "Server hostname for network booting (`sname` field in DHCP packet).",
				Computed:    true,
			},
			"valid_lifetime": schema.Int64Attribute{
				Description: "Lease lifetime in seconds for clients in this class.",
				Computed:    true,
			},
			"min_valid_lifetime": schema.Int64Attribute{
				Description: "Minimum lease lifetime in seconds a client in this class can request.",
				Computed:    true,
			},
			"max_valid_lifetime": schema.Int64Attribute{
				Description: "Maximum lease lifetime in seconds a client in this class can request.",
				Computed:    true,
			},
			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this client class.",
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
			"option_data": optionDataDataSourceAttribute("DHCP options configured for this client class."),
			"option_def": schema.ListNestedAttribute{
				Description: "Custom option definitions of this client class.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Option name.",
							Computed:    true,
						},
						"code": schema.Int32Attribute{
							Description: "Option code.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Option data type.",
							Computed:    true,
						},
						"array": schema.BoolAttribute{
							Description: "Whether the option carries an array of values of `type`.",
							Computed:    true,
						},
						"record_types": schema.StringAttribute{
							Description: "Comma-separated field types when `type` is `record`.",
							Computed:    true,
						},
						"space": schema.StringAttribute{
							Description: "Option space.",
							Computed:    true,
						},
						"encapsulate": schema.StringAttribute{
							Description: "Option space encapsulated by this option.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ClientClassDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClientClassDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ClientClassModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, reportDrift := driftWarnings(ctx, &resp.Diagnostics, &data.ID)
	defer reportDrift()

	class, err := d.client.GetClientClass(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Client Class", err.Error())
		return
	}

	if class == nil {
		resp.Diagnostics.AddError("Client Class Not Found",
			"No client class found with the specified name.")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, class)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
