package keadhcp4

import (
	"context"
	"fmt"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

type Client struct {
	transport           kea.Transport
	configWrite         bool
	configWriteFilename string
}

// ClientOption configures optional behaviour of a Client.
type ClientOption func(*Client)

// WithConfigWrite makes the client send config-write after every successful
// mutating command. An empty filename lets Kea write to the file it was
// started with.
func WithConfigWrite(filename string) ClientOption {
	return func(c *Client) {
		c.configWrite = true
		c.configWriteFilename = filename
	}
}

func NewClient(transport kea.Transport, opts ...ClientOption) *Client {
	c := &Client{transport: transport}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) persist(ctx context.Context, done string) error {
	if !c.configWrite {
		return nil
	}
	if err := c.ConfigWrite(ctx, c.configWriteFilename); err != nil {
		return fmt.Errorf("%s but config-write failed: %w", done, err)
	}
	return nil
}
