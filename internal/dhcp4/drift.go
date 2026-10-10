package dhcp4

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea"
)

// driftWarnings returns a context that collects the servers answering
// differently from the answer used or not answering, and a function that adds
// a warning to diags for each of them. id is read when the function runs.
func driftWarnings(ctx context.Context, diags *diag.Diagnostics, id *types.String) (context.Context, func()) {
	var drifts []kea.Drift
	ctx = kea.WithDriftReporter(ctx, func(d kea.Drift) { drifts = append(drifts, d) })

	return ctx, func() {
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
			diags.AddWarning("Servers Out of Sync",
				fmt.Sprintf("%s answers differently to %s%s; the answer of the first server that answered was used.", d.Server, d.Command, subject))
		}
	}
}
