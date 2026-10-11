package capabilities

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/wfdewith/terraform-provider-kea/internal/errors"
)

type VersionGetter interface {
	Version(context.Context) (*version.Version, error)
}

type Feature struct {
	Name       string
	MinVersion *version.Version
}

type Required interface {
	RequiredFeatures(context.Context) ([]Feature, diag.Diagnostics)
}

func Check(ctx context.Context, versions VersionGetter, required Required, diags *diag.Diagnostics) {
	features, featureDiags := required.RequiredFeatures(ctx)
	diags.Append(featureDiags...)
	if diags.HasError() || len(features) == 0 {
		return
	}

	server, err := versions.Version(ctx)
	if err != nil {
		diags.Append(errors.NewKeaVersionError(err))
		return
	}

	for _, feature := range features {
		if server.LessThan(feature.MinVersion) {
			diags.Append(errors.NewUnsupportedFeatureError(feature.Name, feature.MinVersion.String(), server.String()))
		}
	}
}
