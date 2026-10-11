package kea

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
)

func GetVersion(ctx context.Context, trans Transport) (*version.Version, error) {
	raw, err := ExecWithTextResponse(ctx, trans, "version-get", nil)
	if err != nil {
		return nil, err
	}

	serverVersion, err := version.NewVersion(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", raw, err)
	}

	return serverVersion, nil
}
