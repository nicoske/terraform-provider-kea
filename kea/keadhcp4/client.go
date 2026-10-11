package keadhcp4

import (
	"context"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/wfdewith/terraform-provider-kea/kea"
)

type Client struct {
	transport kea.Transport

	mu      sync.Mutex
	version *version.Version
}

func NewClient(transport kea.Transport) *Client {
	return &Client{transport: transport}
}

func (c *Client) Version(ctx context.Context) (*version.Version, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.version != nil {
		return c.version, nil
	}

	v, err := kea.GetVersion(ctx, c.transport)
	if err != nil {
		return nil, err
	}

	c.version = v
	return v, nil
}
