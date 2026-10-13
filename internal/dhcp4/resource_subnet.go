package dhcp4

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ resource.ResourceWithConfigure = (*SubnetResource)(nil)
var _ resource.ResourceWithImportState = (*SubnetResource)(nil)

type SubnetResource struct {
	client *keadhcp4.Client
}

func (r *SubnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_subnet"
}

func (r *SubnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCPv4 subnet in the Kea DHCP server, including its pools and DHCP options.\n\n" +
			"Every change is saved to the configuration file of the server with `config-write`.\n\n" +
			"Timers left unset (`valid_lifetime`, `min_valid_lifetime`, `max_valid_lifetime`, `renew_timer` and `rebind_timer`) take the value Kea derives from the global configuration.\n\n" +
			"**Important:** Requires the `subnet_cmds` hook library.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Subnet ID. Must be unique within the server. " +
					"Kea only assigns IDs itself when no subnet has an explicit ID, so the ID is always set explicitly.",
				Required: true,
				Validators: []validator.Int64{
					int64validator.Between(1, math.MaxUint32-1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"subnet": schema.StringAttribute{
				Description: "Subnet prefix in CIDR notation (e.g., `192.0.2.0/24`).",
				Required:    true,
				CustomType:  cidrtypes.IPv4PrefixType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"shared_network_name": schema.StringAttribute{
				Description: "Name of the shared network the subnet belongs to, if any. Shared networks are not managed by this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"client_classes": schema.SetAttribute{
				Description: "Restrict the subnet to clients in at least one of these classes.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"evaluate_additional_classes": schema.SetAttribute{
				Description: "Classes evaluated after the subnet is selected (`evaluate-additional-classes`, formerly `require-client-classes`).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"interface": schema.StringAttribute{
				Description: "Name of the interface the subnet is directly reachable on. The interface must exist on the server.",
				Optional:    true,
			},
			"relay": schema.SetAttribute{
				Description: "Addresses of relay agents that forward requests for this subnet (`relay.ip-addresses`).",
				Optional:    true,
				ElementType: iptypes.IPv4AddressType{},
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
			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet).",
				Optional:    true,
			},

			"valid_lifetime": schema.Int64Attribute{
				Description:   "Lease lifetime in seconds. Defaults to the value derived from the global configuration.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.Between(0, math.MaxUint32)},
				PlanModifiers: []planmodifier.Int64{derivedTimerModifier{}},
			},
			"min_valid_lifetime": schema.Int64Attribute{
				Description:   "Minimum lease lifetime in seconds a client may request. Defaults to `valid_lifetime`.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.Between(0, math.MaxUint32)},
				PlanModifiers: []planmodifier.Int64{derivedTimerModifier{}},
			},
			"max_valid_lifetime": schema.Int64Attribute{
				Description:   "Maximum lease lifetime in seconds a client may request. Defaults to `valid_lifetime`.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.Between(0, math.MaxUint32)},
				PlanModifiers: []planmodifier.Int64{derivedTimerModifier{}},
			},
			"renew_timer": schema.Int64Attribute{
				Description:   "T1 timer in seconds (Option 58). Defaults to the value derived from the global configuration.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.Between(0, math.MaxUint32)},
				PlanModifiers: []planmodifier.Int64{derivedTimerModifier{}},
			},
			"rebind_timer": schema.Int64Attribute{
				Description:   "T2 timer in seconds (Option 59). Defaults to the value derived from the global configuration.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.Int64{int64validator.Between(0, math.MaxUint32)},
				PlanModifiers: []planmodifier.Int64{derivedTimerModifier{}},
			},

			"reservations_global": schema.BoolAttribute{
				Description: "Look up global host reservations for clients in this subnet.",
				Optional:    true,
			},
			"reservations_in_subnet": schema.BoolAttribute{
				Description: "Look up host reservations made for this subnet.",
				Optional:    true,
			},
			"reservations_out_of_pool": schema.BoolAttribute{
				Description: "Assume reserved addresses are outside the pools, which speeds up allocation. Only used when `reservations_in_subnet` is `true`.",
				Optional:    true,
			},

			"ddns_send_updates": schema.BoolAttribute{
				Description: "Send DNS updates for leases in this subnet.",
				Optional:    true,
			},
			"ddns_override_no_update": schema.BoolAttribute{
				Description: "Update DNS even when the client asks the server not to.",
				Optional:    true,
			},
			"ddns_override_client_update": schema.BoolAttribute{
				Description: "Update forward DNS even when the client asks to do it itself.",
				Optional:    true,
			},
			"ddns_replace_client_name": schema.StringAttribute{
				Description: "When to replace the name sent by the client: `never`, `always`, `when-present` or `when-not-present`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("never", "always", "when-present", "when-not-present"),
				},
			},
			"ddns_generated_prefix": schema.StringAttribute{
				Description: "Prefix of generated client names.",
				Optional:    true,
			},
			"ddns_qualifying_suffix": schema.StringAttribute{
				Description: "Suffix appended to partial client names.",
				Optional:    true,
			},
			"ddns_update_on_renew": schema.BoolAttribute{
				Description: "Update DNS on every lease renewal, not only when the lease is new or changes.",
				Optional:    true,
			},
			"ddns_ttl": schema.Int64Attribute{
				Description: "TTL in seconds of the DNS records for leases in this subnet.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(0, math.MaxUint32)},
			},

			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this subnet.",
				Optional:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
		},
		Blocks: map[string]schema.Block{
			"pools": schema.ListNestedBlock{
				Description: "Address pools of the subnet. Kea keeps pools sorted by address, so an imported subnet lists them in that order.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"pool": schema.StringAttribute{
							Description: "Address range, either `first - last` (e.g., `192.0.2.10 - 192.0.2.100`) or a prefix (e.g., `192.0.2.128/26`).",
							Required:    true,
							Validators: []validator.String{
								poolValidator{},
							},
						},
						"client_classes": schema.SetAttribute{
							Description: "Restrict the pool to clients in at least one of these classes.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"evaluate_additional_classes": schema.SetAttribute{
							Description: "Classes evaluated after an address is allocated from the pool.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"option_data": optionDataBlock("DHCP options sent to clients that get an address from this pool."),
					},
				},
			},
			"option_data": optionDataBlock("DHCP options sent to clients in this subnet, overriding global options."),
		},
	}
}

func (r *SubnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SubnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SubnetModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var config SubnetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configured := config.configuredTimers()
	data.clearDerivedTimers(configured)

	subnet, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	writeErr := r.client.AddSubnet(ctx, subnet)
	if !applied(writeErr) {
		resp.Diagnostics.AddError("Error Adding Subnet", writeErr.Error())
		return
	}
	defer addConfigWriteError(&resp.Diagnostics, writeErr)

	created, err := r.client.GetSubnet(ctx, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Created Subnet", err.Error())
		return
	}

	if created == nil {
		resp.Diagnostics.AddError("Error Reading Created Subnet", "Subnet not found after creation")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, created)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(setConfiguredTimers(ctx, resp.Private, configured)...)
}

func (r *SubnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SubnetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := r.client.GetSubnet(ctx, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Subnet", err.Error())
		return
	}

	if subnet == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, subnet)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SubnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SubnetModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var config SubnetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configured := config.configuredTimers()
	data.clearDerivedTimers(configured)

	subnet, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	writeErr := r.client.UpdateSubnet(ctx, subnet)
	if !applied(writeErr) {
		resp.Diagnostics.AddError("Error Updating Subnet", writeErr.Error())
		return
	}
	defer addConfigWriteError(&resp.Diagnostics, writeErr)

	updated, err := r.client.GetSubnet(ctx, data.BuildQuery())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Updated Subnet", err.Error())
		return
	}

	if updated == nil {
		resp.Diagnostics.AddError("Error Reading Updated Subnet", "Subnet not found after update")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, updated)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(setConfiguredTimers(ctx, resp.Private, configured)...)
}

func (r *SubnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SubnetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSubnet(ctx, uint32(data.ID.ValueInt64())); err != nil {
		addDeleteError(&resp.Diagnostics, "Error Deleting Subnet", err)
	}
}

func (r *SubnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseUint(req.ID, 10, 32)
	if err != nil || id == 0 || id == math.MaxUint32 {
		resp.Diagnostics.AddError("Invalid Import ID",
			fmt.Sprintf("expected a subnet ID between 1 and %d, got %q", uint32(math.MaxUint32-1), req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), int64(id))...)
}
