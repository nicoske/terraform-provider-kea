package dhcp4

import (
	"context"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ datasource.DataSourceWithConfigure = (*SubnetDataSource)(nil)
var _ datasource.DataSourceWithConfigValidators = (*SubnetDataSource)(nil)

type SubnetDataSource struct {
	client *keadhcp4.Client
}

func (d *SubnetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_subnet"
}

func (d *SubnetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a DHCPv4 subnet from the Kea DHCP server by ID or by prefix. " +
			"Works for subnets from the configuration file as well as subnets managed by `kea_dhcp4_subnet`.\n\n" +
			"**Important:** Requires the `subnet_cmds` hook library.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Subnet ID. Exactly one of `id` and `subnet` must be set.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, math.MaxUint32-1),
				},
			},
			"subnet": schema.StringAttribute{
				Description: "Subnet prefix in CIDR notation, as written in the Kea configuration. Exactly one of `id` and `subnet` must be set.",
				Optional:    true,
				Computed:    true,
				CustomType:  cidrtypes.IPv4PrefixType{},
			},
			"shared_network_name": schema.StringAttribute{
				Description: "Name of the shared network the subnet belongs to, if any.",
				Computed:    true,
			},

			"client_classes": schema.SetAttribute{
				Description: "Classes a client must belong to (at least one) to use the subnet.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"evaluate_additional_classes": schema.SetAttribute{
				Description: "Classes evaluated after the subnet is selected.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"interface": schema.StringAttribute{
				Description: "Name of the interface the subnet is directly reachable on.",
				Computed:    true,
			},
			"relay": schema.SetAttribute{
				Description: "Addresses of relay agents that forward requests for this subnet.",
				Computed:    true,
				ElementType: iptypes.IPv4AddressType{},
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
			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet).",
				Computed:    true,
			},

			"valid_lifetime": schema.Int64Attribute{
				Description: "Lease lifetime in seconds.",
				Computed:    true,
			},
			"min_valid_lifetime": schema.Int64Attribute{
				Description: "Minimum lease lifetime in seconds a client may request.",
				Computed:    true,
			},
			"max_valid_lifetime": schema.Int64Attribute{
				Description: "Maximum lease lifetime in seconds a client may request.",
				Computed:    true,
			},
			"renew_timer": schema.Int64Attribute{
				Description: "T1 timer in seconds (Option 58).",
				Computed:    true,
			},
			"rebind_timer": schema.Int64Attribute{
				Description: "T2 timer in seconds (Option 59).",
				Computed:    true,
			},

			"reservations_global": schema.BoolAttribute{
				Description: "Whether global host reservations are looked up for clients in this subnet.",
				Computed:    true,
			},
			"reservations_in_subnet": schema.BoolAttribute{
				Description: "Whether host reservations made for this subnet are looked up.",
				Computed:    true,
			},
			"reservations_out_of_pool": schema.BoolAttribute{
				Description: "Whether reserved addresses are assumed to be outside the pools.",
				Computed:    true,
			},

			"ddns_send_updates": schema.BoolAttribute{
				Description: "Whether DNS updates are sent for leases in this subnet.",
				Computed:    true,
			},
			"ddns_override_no_update": schema.BoolAttribute{
				Description: "Whether DNS is updated even when the client asks the server not to.",
				Computed:    true,
			},
			"ddns_override_client_update": schema.BoolAttribute{
				Description: "Whether forward DNS is updated even when the client asks to do it itself.",
				Computed:    true,
			},
			"ddns_replace_client_name": schema.StringAttribute{
				Description: "When the name sent by the client is replaced.",
				Computed:    true,
			},
			"ddns_generated_prefix": schema.StringAttribute{
				Description: "Prefix of generated client names.",
				Computed:    true,
			},
			"ddns_qualifying_suffix": schema.StringAttribute{
				Description: "Suffix appended to partial client names.",
				Computed:    true,
			},
			"ddns_update_on_renew": schema.BoolAttribute{
				Description: "Whether DNS is updated on every lease renewal.",
				Computed:    true,
			},
			"ddns_ttl": schema.Int64Attribute{
				Description: "TTL in seconds of the DNS records for leases in this subnet.",
				Computed:    true,
			},

			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this subnet.",
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
			"pools": schema.ListNestedAttribute{
				Description: "Address pools of the subnet, sorted by address.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pool": schema.StringAttribute{
							Description: "Address range, as Kea reports it (`first-last` or a prefix).",
							Computed:    true,
						},
						"client_classes": schema.SetAttribute{
							Description: "Classes a client must belong to (at least one) to use the pool.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"evaluate_additional_classes": schema.SetAttribute{
							Description: "Classes evaluated after an address is allocated from the pool.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"option_data": optionDataDataSourceAttribute("DHCP options configured for this pool."),
					},
				},
			},
			"option_data": optionDataDataSourceAttribute("DHCP options configured for this subnet."),
		},
	}
}

func (d *SubnetDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("subnet"),
		),
	}
}

func (d *SubnetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SubnetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SubnetModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := d.client.GetSubnet(ctx, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Subnet", err.Error())
		return
	}

	if subnet == nil {
		resp.Diagnostics.AddError("Subnet Not Found",
			"No subnet found matching the specified criteria.")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, subnet)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
