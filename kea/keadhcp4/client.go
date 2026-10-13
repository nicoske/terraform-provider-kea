package keadhcp4

import (
	"context"
	"fmt"
	"sync"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

type Client struct {
	transport           kea.Transport
	configWriteFilename string

	targetMu sync.Mutex
	target   kea.OperationTarget
}

// ClientOption configures optional behaviour of a Client.
type ClientOption func(*Client)

// WithConfigWriteFilename makes config-write write to filename instead of the
// file Kea was started with.
func WithConfigWriteFilename(filename string) ClientOption {
	return func(c *Client) {
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

// ConfigWriteError reports a command that Kea applied to its running
// configuration but could not write to disk.
type ConfigWriteError struct {
	Done string
	Err  error
}

func (e *ConfigWriteError) Error() string {
	return fmt.Sprintf("%s but config-write failed: %v", e.Done, e.Err)
}

func (e *ConfigWriteError) Unwrap() error {
	return e.Err
}

// exec sends a command that changes the configuration, then config-write so
// that Kea keeps the change across restarts.
func (c *Client) exec(ctx context.Context, done, command string, arguments any) error {
	if err := kea.Exec(ctx, c.transport, command, arguments); err != nil {
		return err
	}
	if err := c.ConfigWrite(ctx, c.configWriteFilename); err != nil {
		return &ConfigWriteError{Done: done, Err: err}
	}
	return nil
}

// ReservationTarget returns the operation target for reservation commands:
// the hosts database when the server has one, otherwise its running
// configuration, which config-write saves to disk.
func (c *Client) ReservationTarget(ctx context.Context) (kea.OperationTarget, error) {
	c.targetMu.Lock()
	defer c.targetMu.Unlock()

	if c.target != "" {
		return c.target, nil
	}

	hasDatabase, err := c.hasHostsDatabase(ctx)
	if err != nil {
		return "", fmt.Errorf("detecting hosts database: %w", err)
	}

	c.target = kea.OperationTargetMemory
	if hasDatabase {
		c.target = kea.OperationTargetDatabase
	}
	return c.target, nil
}
