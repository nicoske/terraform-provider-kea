package dhcp4

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

func describes(want, have keadhcp4.OptionData) bool {
	return optionalEqual(want.Name, have.Name) &&
		optionalEqual(want.Code, have.Code) &&
		optionalEqual(want.Space, have.Space) &&
		optionalEqual(want.AlwaysSend, have.AlwaysSend) &&
		optionalEqual(want.NeverSend, have.NeverSend) &&
		kea.ValueDescribes(want.Value, have.Value) &&
		sameClasses(want.ClientClasses, have.ClientClasses)
}

func optionalEqual[T comparable](a, b *T) bool {
	return a == nil || (b != nil && *a == *b)
}

func sameClasses(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// matchOptions returns, for each option, the index of a distinct model that
// describes it, or -1. It finds a maximum bipartite matching, so the result
// does not depend on iteration order.
func matchOptions(ctx context.Context, models []OptionDataModel, options []keadhcp4.OptionData) ([]int, diag.Diagnostics) {
	var diags diag.Diagnostics
	parsed := make([]keadhcp4.OptionData, len(models))
	for i := range models {
		od, d := models[i].ToAPI(ctx)
		diags.Append(d...)
		parsed[i] = od
	}

	available := make([]bool, len(models))
	for j := range models {
		available[j] = !models[j].Data.IsUnknown()
	}

	matchedHave := make([]int, len(models))
	for j := range matchedHave {
		matchedHave[j] = -1
	}

	var augment func(i int, seen []bool) bool
	augment = func(i int, seen []bool) bool {
		for j := range parsed {
			if !available[j] || seen[j] || !describes(parsed[j], options[i]) {
				continue
			}
			seen[j] = true
			if matchedHave[j] == -1 || augment(matchedHave[j], seen) {
				matchedHave[j] = i
				return true
			}
		}
		return false
	}

	for i := range options {
		augment(i, make([]bool, len(models)))
	}

	modelIndex := make([]int, len(options))
	for i := range modelIndex {
		modelIndex[i] = -1
	}
	for j, i := range matchedHave {
		if i != -1 {
			modelIndex[i] = j
		}
	}

	return modelIndex, diags
}
