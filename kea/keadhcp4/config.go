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

func (c *Client) hasHostsDatabase(ctx context.Context) (bool, error) {
	result, err := kea.ExecWithResponse[struct {
		Dhcp4 struct {
			HostsDatabase  map[string]any   `json:"hosts-database"`
			HostsDatabases []map[string]any `json:"hosts-databases"`
		} `json:"Dhcp4"`
	}](ctx, c.transport, "config-get", struct{}{})

	if err != nil {
		return false, err
	}

	if result == nil {
		return false, nil
	}

	return len(result.Dhcp4.HostsDatabase) > 0 || len(result.Dhcp4.HostsDatabases) > 0, nil
}
