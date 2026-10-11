package capabilities

import "github.com/hashicorp/go-version"

var (
	// OptionDataClientClasses requires Kea 3.1.0. Kea 3.0.x silently drops
	// option-data client-classes in host reservations (GitLab #3770).
	OptionDataClientClasses = Feature{
		Name:       "option_data.client_classes",
		MinVersion: version.Must(version.NewVersion("3.1.0")),
	}
)
