package dhcp4

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea"
)

// driftWarnings returns a context that collects the servers answering
// differently from the answer used or not answering, and a function that adds
// a warning to diags for each of them and reports whether every server that
// answered gave the same answer. id is read when the function runs.
func driftWarnings(ctx context.Context, diags *diag.Diagnostics, id *types.String) (context.Context, func() bool) {
	var drifts []kea.Drift
	ctx = kea.WithDriftReporter(ctx, func(d kea.Drift) { drifts = append(drifts, d) })

	return ctx, func() bool {
		inSync := true
		for _, d := range drifts {
			var subject string
			if id.ValueString() != "" {
				subject = " for " + id.ValueString()
			}
			if d.Unavailable() {
				diags.AddWarning("Server Unavailable",
					fmt.Sprintf("%s did not answer %s%s: %v; the answer of the other servers was used.", d.Server, d.Command, subject, d.Err))
				continue
			}
			inSync = false
			diags.AddWarning("Servers Out of Sync",
				fmt.Sprintf("%s answers differently to %s%s; the answer of the first server that answered was used.", d.Server, d.Command, subject))
		}
		return inSync
	}
}

func inSyncAttribute() schema.BoolAttribute {
	return schema.BoolAttribute{
		Computed: true,
		Description: "Whether every server of the provider answers the same for this resource. " +
			"`false` plans an update that writes it to every server again.",
		PlanModifiers: []planmodifier.Bool{resyncModifier{}},
	}
}

// resyncModifier plans in_sync true when the state holds false, so that
// Terraform updates the resource, and keeps the state value otherwise.
type resyncModifier struct{}

func (m resyncModifier) Description(_ context.Context) string {
	return "Plans an update when the servers answer differently."
}

func (m resyncModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m resyncModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() {
		return
	}
	if !req.StateValue.ValueBool() {
		resp.PlanValue = types.BoolValue(true)
		return
	}
	if resp.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}
