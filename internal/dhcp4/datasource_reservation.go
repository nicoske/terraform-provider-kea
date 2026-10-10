package dhcp4

import (
	"context"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
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
	"github.com/wfdewith/terraform-provider-kea/internal/keatypes"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ datasource.DataSourceWithConfigure = (*ReservationDataSource)(nil)
var _ datasource.DataSourceWithConfigValidators = (*ReservationDataSource)(nil)

type ReservationDataSource struct {
	client *keadhcp4.Client
}

func (d *ReservationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_reservation"
}

func (d *ReservationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a DHCPv4 host reservation from the Kea DHCP server.\n\n" +
			"**Important:** Requires the `host_cmds` hook library.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the reservation in the format `{subnet_id}/{identifier_type}/{identifier}`.",
			},
			"subnet_id": schema.Int64Attribute{
				Description: "Subnet ID for this reservation, or `0` for a global reservation.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, math.MaxUint32),
				},
			},

			"circuit_id": schema.StringAttribute{
				Description: "Circuit ID (Option 82 sub-option 1) to identify the client. Typically inserted by relay agents. " +
					"Mutually exclusive with other identifier types.",
				Optional:   true,
				CustomType: keatypes.HexIDType{},
			},
			"client_id": schema.StringAttribute{
				Description: "Client identifier (Option 61) to identify the client. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  keatypes.HexIDType{},
			},
			"duid": schema.StringAttribute{
				Description: "DHCP Unique Identifier. Typically used in DHCPv6 but also supported for DHCPv4. " +
					"Mutually exclusive with other identifier types.",
				Optional:   true,
				CustomType: keatypes.HexIDType{},
			},
			"flex_id": schema.StringAttribute{
				Description: "Flexible identifier from the `flex_id` hook library. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  keatypes.HexIDType{},
			},
			"hw_address": schema.StringAttribute{
				Description: "Hardware (MAC) address to identify the client. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  hwtypes.MACAddressType{},
			},

			"ip_address": schema.StringAttribute{
				Description: "Reserved IPv4 address. Can be used to query for a reservation by IP address. " +
					"Mutually exclusive with other identifier types.",
				Optional:   true,
				Computed:   true,
				CustomType: iptypes.IPv4AddressType{},
			},

			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet, Option 67).",
				Computed:    true,
			},
			"client_classes": schema.SetAttribute{
				Description: "Assigned client classes.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"hostname": schema.StringAttribute{
				Description: "Assigned hostname.",
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
			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this reservation.",
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
			"operation_target": schema.StringAttribute{
				Description: "Where Kea looks for the reservation: `database` for the hosts database, " +
					"`memory` for the running server configuration, or `all` (default) for both.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					operationTargetValidator(),
				},
			},
			"option_data": optionDataDataSourceAttribute("DHCP options configured for this reservation."),
		},
	}
}

func (d *ReservationDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("circuit_id"),
			path.MatchRoot("client_id"),
			path.MatchRoot("duid"),
			path.MatchRoot("flex_id"),
			path.MatchRoot("hw_address"),
			path.MatchRoot("ip_address"),
		),
	}
}

func (d *ReservationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ReservationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target := data.Target(kea.OperationTargetAll)
	data.OperationTarget = types.StringValue(string(target))

	reservation, err := d.client.GetReservation(ctx, target, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Reservation", err.Error())
		return
	}

	if reservation == nil {
		resp.Diagnostics.AddError("Reservation Not Found",
			"No reservation found matching the specified criteria.")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, reservation)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
