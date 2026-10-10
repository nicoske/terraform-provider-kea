package dhcp4

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Kea derives the subnet timers left unset from the timers that are set and
// from the global configuration.
var subnetTimers = []string{"valid_lifetime", "min_valid_lifetime", "max_valid_lifetime", "renew_timer", "rebind_timer"}

// configuredTimersKey holds in private state the timers set in the
// configuration of the last apply.
const configuredTimersKey = "configured_timers"

func (m *SubnetModel) timers() map[string]*types.Int64 {
	return map[string]*types.Int64{
		"valid_lifetime":     &m.ValidLifetime,
		"min_valid_lifetime": &m.MinValidLifetime,
		"max_valid_lifetime": &m.MaxValidLifetime,
		"renew_timer":        &m.RenewTimer,
		"rebind_timer":       &m.RebindTimer,
	}
}

// configuredTimers returns the timers that m, read from the configuration,
// sets.
func (m *SubnetModel) configuredTimers() []string {
	timers := m.timers()
	var configured []string
	for _, name := range subnetTimers {
		if !timers[name].IsNull() {
			configured = append(configured, name)
		}
	}
	return configured
}

// clearDerivedTimers sets the timers missing from configured to null, so that
// Kea derives them again instead of keeping the planned value.
func (m *SubnetModel) clearDerivedTimers(configured []string) {
	for name, timer := range m.timers() {
		if !slices.Contains(configured, name) {
			*timer = types.Int64Null()
		}
	}
}

type privateState interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func setConfiguredTimers(ctx context.Context, private privateState, configured []string) diag.Diagnostics {
	if configured == nil {
		configured = []string{}
	}
	value, err := json.Marshal(configured)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error Saving Configured Timers", err.Error())
		return diags
	}
	return private.SetKey(ctx, configuredTimersKey, value)
}

// derivedTimerModifier keeps the state value of a timer left unset while Kea
// derives it from the same values: no timer set in the last apply is unset,
// and every timer set equals its state value. Otherwise the timer is unknown
// until Kea derives it again.
type derivedTimerModifier struct{}

func (m derivedTimerModifier) Description(_ context.Context) string {
	return "Keeps the derived value of an unset timer while the timers it derives from do not change."
}

func (m derivedTimerModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m derivedTimerModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() || !req.ConfigValue.IsNull() || !resp.PlanValue.IsUnknown() {
		return
	}

	value, diags := req.Private.GetKey(ctx, configuredTimersKey)
	resp.Diagnostics.Append(diags...)
	if value == nil {
		return
	}
	var configured []string
	if err := json.Unmarshal(value, &configured); err != nil {
		return
	}

	for _, name := range subnetTimers {
		var config, state types.Int64
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(name), &config)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(name), &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if config.IsNull() && slices.Contains(configured, name) {
			return
		}
		if !config.IsNull() && !config.Equal(state) {
			return
		}
	}

	resp.PlanValue = req.StateValue
}
