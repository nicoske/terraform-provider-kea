package dhcp4

import (
	"context"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/capabilities"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/internal/keatypes"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ resource.ResourceWithConfigure = (*ReservationResource)(nil)
var _ resource.ResourceWithConfigValidators = (*ReservationResource)(nil)
var _ resource.ResourceWithModifyPlan = (*ReservationResource)(nil)
var _ resource.ResourceWithImportState = (*ReservationResource)(nil)

type ReservationResource struct {
	client *keadhcp4.Client
}

func (r *ReservationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_reservation"
}

func (r *ReservationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCPv4 host reservation in the Kea DHCP server. " +
			"Host reservations bind specific IP addresses and DHCP options to clients identified by MAC address, client ID, or other identifiers.\n\n" +
			"**Important:** Requires the `host_cmds` hook library and a hosts database backend.",
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
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"circuit_id": schema.StringAttribute{
				Description: "Circuit ID (Option 82 sub-option 1) to identify the client. Typically inserted by relay agents. " +
					"Mutually exclusive with other identifier types.",
				Optional:   true,
				CustomType: keatypes.HexIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "Client identifier (Option 61) to identify the client. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  keatypes.HexIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"duid": schema.StringAttribute{
				Description: "DHCP Unique Identifier. Typically used in DHCPv6 but also supported for DHCPv4. " +
					"Mutually exclusive with other identifier types.",
				Optional:   true,
				CustomType: keatypes.HexIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"flex_id": schema.StringAttribute{
				Description: "Flexible identifier from the `flex_id` hook library. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  keatypes.HexIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"hw_address": schema.StringAttribute{
				Description: "Hardware (MAC) address to identify the client. Mutually exclusive with other identifier types.",
				Optional:    true,
				CustomType:  hwtypes.MACAddressType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},

			"ip_address": schema.StringAttribute{
				Description: "IPv4 address to reserve for this client.",
				Optional:    true,
				CustomType:  iptypes.IPv4AddressType{},
			},

			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet, Option 67).",
				Optional:    true,
			},
			"client_classes": schema.SetAttribute{
				Description: "Client classes to assign to this client.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"hostname": schema.StringAttribute{
				Description: "Hostname to assign to the client.",
				Optional:    true,
			},
			"next_server": schema.StringAttribute{
				Description: "Next server address for network booting (`siaddr` field in DHCP packet).",
				Optional:    true,
				CustomType:  iptypes.IPv4AddressType{},
			},
			"server_hostname": schema.StringAttribute{
				Description: "Server hostname for network booting (`sname` field in DHCP packet).",
				Optional:    true,
			},
			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this reservation.",
				Optional:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
		},
		Blocks: map[string]schema.Block{
			"option_data": schema.SetNestedBlock{
				Description: "DHCP options to send to this client, overriding subnet and global options.",
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
						optionPayloadValidator{},
						objectvalidator.AtLeastOneOf(
							path.MatchRoot("name"),
							path.MatchRoot("code"),
						),
					},
				},
			},
		},
	}
}

func (r *ReservationResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("circuit_id"),
			path.MatchRoot("client_id"),
			path.MatchRoot("duid"),
			path.MatchRoot("flex_id"),
			path.MatchRoot("hw_address"),
		),
	}
}

func (r *ReservationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = keaClients.DHCP4
}

func (r *ReservationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var data ReservationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	capabilities.Check(ctx, r.client, &data, &resp.Diagnostics)
}

func (r *ReservationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	capabilities.Check(ctx, r.client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	reservation, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AddReservation(ctx, kea.OperationTargetDatabase, reservation); err != nil {
		resp.Diagnostics.AddError("Error Adding Reservation", err.Error())
		return
	}

	created, err := r.client.GetReservation(ctx, kea.OperationTargetDatabase, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Created Reservation", err.Error())
		return
	}

	if created == nil {
		resp.Diagnostics.AddError("Error Reading Created Reservation", "Reservation not found after creation")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, created)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReservationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reservation, err := r.client.GetReservation(ctx, kea.OperationTargetDatabase, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Reservation", err.Error())
		return
	}

	if reservation == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, reservation)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReservationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	capabilities.Check(ctx, r.client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	reservation, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateReservation(ctx, kea.OperationTargetDatabase, reservation); err != nil {
		resp.Diagnostics.AddError("Error Updating Reservation", err.Error())
		return
	}

	updated, err := r.client.GetReservation(ctx, kea.OperationTargetDatabase, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Updated Reservation", err.Error())
		return
	}

	if updated == nil {
		resp.Diagnostics.AddError("Error Reading Updated Reservation", "Reservation not found after update")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, updated)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReservationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteReservation(ctx, kea.OperationTargetDatabase, data.BuildQuery()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Reservation", err.Error())
	}
}

func (r *ReservationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	fields, diag := ParseReservationID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}
