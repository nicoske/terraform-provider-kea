package keadhcp4

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// mutate runs send, which sends a command that changes the configuration, and
// then config-write when it is enabled. When send stops part way through the
// servers of a kea.MultiTransport, config-write still goes to the servers that
// took the change, so that they keep it across a restart.
func (c *Client) mutate(ctx context.Context, done string, send func() error) error {
	err := send()
	var partial *kea.PartialWriteError
	if errors.As(err, &partial) {
		return c.persistPartial(ctx, err, partial.Done)
	}
	if err != nil {
		return err
	}
	if !c.configWrite {
		return nil
	}
	if err := c.ConfigWrite(ctx, c.configWriteFilename); err != nil {
		return fmt.Errorf("%s but config-write failed: %w", done, err)
	}
	return nil
}

func (c *Client) persistPartial(ctx context.Context, err error, done []string) error {
	multi, ok := c.transport.(*kea.MultiTransport)
	if !c.configWrite || !ok || len(done) == 0 {
		return err
	}
	names := strings.Join(done, ", ")
	if werr := NewClient(multi.Subset(done...)).ConfigWrite(ctx, c.configWriteFilename); werr != nil {
		return fmt.Errorf("%w; config-write on %s failed: %w", err, names, werr)
	}
	return fmt.Errorf("%w; configuration written on %s", err, names)
}

// exec sends a command that changes the configuration through mutate.
func (c *Client) exec(ctx context.Context, done, command string, arguments any) error {
	return c.mutate(ctx, done, func() error {
		return kea.Exec(ctx, c.transport, command, arguments)
	})
}

// servers returns the servers behind a kea.MultiTransport. Any other
// transport is one server with an empty name.
func (c *Client) servers() []kea.Server {
	if multi, ok := c.transport.(*kea.MultiTransport); ok {
		return multi.Servers
	}
	return []kea.Server{{Transport: c.transport}}
}

// upsert sends update to every server that holds the object and add to the
// others, so that a server that lost the object gets it back. exists asks
// one server whether it holds the object.
func (c *Client) upsert(ctx context.Context, add, update string, arguments any, exists func(*Client) (bool, error)) error {
	servers := c.servers()
	var done []string
	for _, server := range servers {
		err := upsertOn(ctx, server, add, update, arguments, exists)
		if err != nil && len(servers) == 1 {
			return err
		}
		if err != nil {
			return &kea.PartialWriteError{Command: update, Done: done, Failed: server.Name, Err: err}
		}
		done = append(done, server.Name)
	}
	return nil
}

func upsertOn(ctx context.Context, server kea.Server, add, update string, arguments any, exists func(*Client) (bool, error)) error {
	found, err := exists(NewClient(server))
	if err != nil {
		return err
	}
	if !found {
		return kea.Exec(ctx, server, add, arguments)
	}
	return kea.Exec(ctx, server, update, arguments)
}
