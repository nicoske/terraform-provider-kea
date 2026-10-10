package dhcp4

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/internal/keatypes"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ resource.ResourceWithConfigure = (*ReservationResource)(nil)
var _ resource.ResourceWithConfigValidators = (*ReservationResource)(nil)
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
			"**Important:** Requires the `host_cmds` hook library, and a hosts database backend unless `operation_target` is `memory`.",
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

			"operation_target": schema.StringAttribute{
				Description: "Where Kea stores the reservation: `database` (default) for the hosts database, " +
					"`memory` for the running server configuration, or `all` for both. " +
					"Reservations in `memory` are lost when Kea restarts unless the configuration is written to disk. " +
					"Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(string(kea.OperationTargetDatabase)),
				Validators: []validator.String{
					operationTargetValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"option_data": optionDataBlock("DHCP options to send to this client, overriding subnet and global options."),
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

func (r *ReservationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, target := req.ID, kea.OperationTargetDatabase
	if prefix, rest, ok := strings.Cut(id, "/"); ok && strings.Count(rest, "/") == 2 {
		id, target = rest, kea.OperationTarget(prefix)
		if !slices.Contains(operationTargets, target) {
			resp.Diagnostics.AddError("Invalid Import ID",
				fmt.Sprintf("unsupported operation target %q, expected one of database, memory or all", prefix))
			return
		}
	}

	subnetID, identifierType, identifier, err := parseReservationID(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	var value attr.Value
	switch identifierType {
	case "hw-address":
		value = hwtypes.NewMACAddressValue(identifier)
	case "ip-address":
		value = iptypes.NewIPv4AddressValue(identifier)
	default:
		value = keatypes.NewHexIDValue(identifier)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operation_target"), string(target))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subnet_id"), subnetID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(strings.ReplaceAll(identifierType, "-", "_")), value)...)
}

func (r *ReservationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ReservationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reservation, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AddReservation(ctx, data.Target(kea.OperationTargetDatabase), reservation); err != nil {
		resp.Diagnostics.AddError("Error Adding Reservation", err.Error())
		return
	}

	created, err := r.client.GetReservation(ctx, data.Target(kea.OperationTargetDatabase), data.BuildQuery())
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

	// null in state written by provider versions without operation_target
	if data.OperationTarget.IsNull() {
		data.OperationTarget = types.StringValue(string(kea.OperationTargetDatabase))
	}

	reservation, err := r.client.GetReservation(ctx, data.Target(kea.OperationTargetDatabase), data.BuildQuery())
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

	reservation, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateReservation(ctx, data.Target(kea.OperationTargetDatabase), reservation); err != nil {
		resp.Diagnostics.AddError("Error Updating Reservation", err.Error())
		return
	}

	updated, err := r.client.GetReservation(ctx, data.Target(kea.OperationTargetDatabase), data.BuildQuery())
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

	if err := r.client.DeleteReservation(ctx, data.Target(kea.OperationTargetDatabase), data.BuildQuery()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Reservation", err.Error())
	}
}
