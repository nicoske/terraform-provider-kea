package errors

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func NewProviderDataTypeError(value any) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unexpected ProviderData Type",
		fmt.Sprintf(
			"Expected clients.KeaClients, got: %T. "+
				"Please report this issue to the provider developers.",
			value,
		),
	)
}

func NewUnconfiguredClientError(client string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unconfigured Client Error",
		fmt.Sprintf(
			"Expected Kea %s client, but it was not configured. "+
				"Please configure the '%s' block in the provider configuration.",
			strings.ToUpper(client),
			client,
		),
	)
}

func NewKeaVersionError(err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Error Determining Kea Version",
		fmt.Sprintf(
			"Unable to determine the Kea server version: %s. "+
				"The version is required to validate this configuration.",
			err,
		),
	)
}

func NewUnsupportedFeatureError(feature, minVersion, serverVersion string) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unsupported Kea Version",
		fmt.Sprintf(
			"The %s attribute requires Kea %s or newer; the server reports %s. "+
				"Upgrade Kea or remove the attribute.",
			feature,
			minVersion,
			serverVersion,
		),
	)
}
