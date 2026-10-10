package dhcp4

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/wfdewith/terraform-provider-kea/kea"
)

// driftWarnings returns a context that collects the servers answering
// differently from the first server, and a function that adds a warning to
// diags for each of them. id is read when the function runs.
func driftWarnings(ctx context.Context, diags *diag.Diagnostics, id *types.String) (context.Context, func()) {
	var drifts []kea.Drift
	ctx = kea.WithDriftReporter(ctx, func(d kea.Drift) { drifts = append(drifts, d) })

	return ctx, func() {
		for _, d := range drifts {
			detail := fmt.Sprintf("%s answers differently to %s", d.Server, d.Command)
			if id.ValueString() != "" {
				detail += " for " + id.ValueString()
			}
			diags.AddWarning("Servers Out of Sync", detail+"; the first server's answer was used.")
		}
	}
}
