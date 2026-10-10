package dhcp4

import (
	"context"
	"math"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

var _ resource.ResourceWithConfigure = (*ClientClassResource)(nil)
var _ resource.ResourceWithConfigValidators = (*ClientClassResource)(nil)
var _ resource.ResourceWithImportState = (*ClientClassResource)(nil)

type ClientClassResource struct {
	client *keadhcp4.Client
}

func (r *ClientClassResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp4_client_class"
}

func (r *ClientClassResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCPv4 client class in the Kea DHCP server. " +
			"Client classes group clients by a test expression and assign options, boot parameters and lease lifetimes to them.\n\n" +
			"Kea evaluates client classes in the order they are defined, and `class-add` appends a new class at the end of the list. " +
			"A class created by this resource is therefore evaluated after the classes from the configuration file and after the classes created before it. " +
			"A class can only refer to classes defined before it, so use references or `depends_on` to create classes in the right order.\n\n" +
			"**Important:** Requires the `class_cmds` hook library. Kea keeps classes added through `class_cmds` in memory only.",
		Attributes: map[string]schema.Attribute{
			"in_sync": inSyncAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier for the client class, equal to its name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the client class.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"test": schema.StringAttribute{
				Description: "Expression that a client must match to belong to this class. " +
					"Kea rejects an update that changes whether the class depends on the `KNOWN` or `UNKNOWN` built-in classes. " +
					"Conflicts with `template_test`.",
				Optional: true,
			},
			"template_test": schema.StringAttribute{
				Description: "Expression whose result names a subclass the client is assigned to, making this class a template class. " +
					"Removing it replaces the class, since Kea does not turn a template class back into a regular class. " +
					"Conflicts with `test`.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
						},
						"Removing template_test requires replacement.",
						"Removing `template_test` requires replacement.",
					),
				},
			},
			"only_in_additional_list": schema.BoolAttribute{
				Description: "When `true`, the class is only evaluated when a subnet, pool or shared network lists it in `evaluate-additional-classes`. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"boot_file_name": schema.StringAttribute{
				Description: "Boot file name for network booting (`file` field in DHCP packet, Option 67).",
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
			"valid_lifetime": schema.Int64Attribute{
				Description: "Lease lifetime in seconds for clients in this class.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, math.MaxUint32),
				},
			},
			"min_valid_lifetime": schema.Int64Attribute{
				Description: "Minimum lease lifetime in seconds a client in this class can request.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, math.MaxUint32),
				},
			},
			"max_valid_lifetime": schema.Int64Attribute{
				Description: "Maximum lease lifetime in seconds a client in this class can request.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, math.MaxUint32),
				},
			},
			"user_context": schema.StringAttribute{
				Description: "Arbitrary JSON data stored with this client class.",
				Optional:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
		},
		Blocks: map[string]schema.Block{
			"option_data": optionDataBlock("DHCP options to send to clients in this class."),
			"option_def": schema.ListNestedBlock{
				Description: "Custom option definitions for this class.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Option name.",
							Required:    true,
						},
						"code": schema.Int32Attribute{
							Description: "Option code (0-255).",
							Required:    true,
							Validators: []validator.Int32{
								int32validator.Between(0, math.MaxUint8),
							},
						},
						"type": schema.StringAttribute{
							Description: "Option data type (e.g., `uint8`, `string`, `ipv4-address`, `record`).",
							Required:    true,
						},
						"array": schema.BoolAttribute{
							Description: "Whether the option carries an array of values of `type`. Defaults to `false`.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"record_types": schema.StringAttribute{
							Description: "Comma-separated field types when `type` is `record`.",
							Optional:    true,
						},
						"space": schema.StringAttribute{
							Description: "Option space. Defaults to `dhcp4`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("dhcp4"),
						},
						"encapsulate": schema.StringAttribute{
							Description: "Option space encapsulated by this option.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *ClientClassResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("test"),
			path.MatchRoot("template_test"),
		),
	}
}

func (r *ClientClassResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClientClassResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClientClassResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	class, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AddClientClass(ctx, class); err != nil {
		resp.Diagnostics.AddError("Error Adding Client Class", err.Error())
		return
	}

	created, err := r.client.GetClientClass(ctx, class.Name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Created Client Class", err.Error())
		return
	}

	if created == nil {
		resp.Diagnostics.AddError("Error Reading Created Client Class", "Client class not found after creation")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, created)...)
	data.InSync = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClientClassResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClientClassResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, reportDrift := driftWarnings(ctx, &resp.Diagnostics, &data.ID)

	class, err := r.client.GetClientClass(ctx, data.Name.ValueString())
	data.InSync = types.BoolValue(reportDrift())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Client Class", err.Error())
		return
	}

	if class == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, class)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClientClassResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClientClassResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	class, diags := data.ToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateClientClass(ctx, class); err != nil {
		resp.Diagnostics.AddError("Error Updating Client Class", err.Error())
		return
	}

	updated, err := r.client.GetClientClass(ctx, class.Name)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Updated Client Class", err.Error())
		return
	}

	if updated == nil {
		resp.Diagnostics.AddError("Error Reading Updated Client Class", "Client class not found after update")
		return
	}

	resp.Diagnostics.Append(data.FromAPI(ctx, updated)...)
	data.InSync = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClientClassResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClientClassResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteClientClass(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Deleting Client Class", err.Error())
	}
}

func (r *ClientClassResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
