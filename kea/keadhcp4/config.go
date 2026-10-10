package keadhcp4

import (
	"context"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

// ConfigWrite writes the running configuration to filename, or to the file
// Kea was started with when filename is empty. Kea only accepts files in the
// directory of its startup configuration file.
func (c *Client) ConfigWrite(ctx context.Context, filename string) error {
	return kea.Exec(ctx, c.transport, "config-write", struct {
		Filename string `json:"filename,omitempty"`
	}{filename})
}

// ConfigHash returns the SHA-256 hash of the running configuration.
func (c *Client) ConfigHash(ctx context.Context) (string, error) {
	result, err := kea.ExecWithResponse[struct {
		Hash string `json:"hash"`
	}](ctx, c.transport, "config-hash-get", struct{}{})

	if err != nil {
		return "", err
	}

	if result == nil {
		return "", nil
	}

	return result.Hash, nil
}
