package kea

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

var _ Transport = &MultiTransport{}

// Server is a Transport to one Kea server, with the name used in errors and
// drift reports.
type Server struct {
	Name string
	Transport
}

// MultiTransport sends every command to several Kea servers that must hold the
// same configuration, such as the peers of a high availability pair, which do
// not replicate configuration changes to each other.
//
// A write goes to every server in order and stops at the first failure with a
// *PartialWriteError. A read goes to every server too and returns the first
// server's answer; a server that answers differently is reported to the
// function set with WithDriftReporter, or fails the read with a *DriftError
// when Strict is set. A single server is used as is.
type MultiTransport struct {
	Servers []Server
	Strict  bool
}

// PartialWriteError reports a write command that failed on server Failed after
// it succeeded on the servers in Done.
type PartialWriteError struct {
	Command string
	Done    []string
	Failed  string
	Err     error
}

func (e *PartialWriteError) Error() string {
	if len(e.Done) == 0 {
		return fmt.Sprintf("%s failed on %s: %v", e.Command, e.Failed, e.Err)
	}
	return fmt.Sprintf("%s succeeded on %s and failed on %s: %v",
		e.Command, strings.Join(e.Done, ", "), e.Failed, e.Err)
}

func (e *PartialWriteError) Unwrap() error {
	return e.Err
}

// Drift is a server whose answer to a read command differs from the first
// server's answer.
type Drift struct {
	Server  string
	Command string
}

// DriftError is returned by a MultiTransport with Strict set when a server
// answers a read command differently from the first server.
type DriftError struct {
	Drift
}

func (e *DriftError) Error() string {
	return fmt.Sprintf("servers out of sync: %s answers differently to %s", e.Server, e.Command)
}

type driftReporterKey struct{}

// WithDriftReporter returns a context that makes a MultiTransport call report
// for every server that answers a read command differently from the first
// server.
func WithDriftReporter(ctx context.Context, report func(Drift)) context.Context {
	return context.WithValue(ctx, driftReporterKey{}, report)
}

var readCommands = []string{
	"config-get",
	"config-hash-get",
	"config-test",
	"list-commands",
	"status-get",
	"version-get",
}

// IsReadCommand reports whether command only reads from Kea.
func IsReadCommand(command string) bool {
	return strings.HasSuffix(command, "-get") ||
		strings.Contains(command, "-get-") ||
		strings.HasSuffix(command, "-list") ||
		slices.Contains(readCommands, command)
}

func (t *MultiTransport) Send(ctx context.Context, req CommandRequest, resp *CommandResponse) error {
	if len(t.Servers) == 1 {
		return t.Servers[0].Send(ctx, req, resp)
	}
	if resp == nil {
		resp = &CommandResponse{}
	}
	if IsReadCommand(req.Command) {
		return t.read(ctx, req, resp)
	}
	return t.write(ctx, req, resp)
}

func (t *MultiTransport) write(ctx context.Context, req CommandRequest, resp *CommandResponse) error {
	var done []string
	for _, server := range t.Servers {
		var r CommandResponse
		err := server.Send(ctx, req, &r)
		if err == nil {
			err = resultError(&r)
		}
		if err != nil {
			return &PartialWriteError{Command: req.Command, Done: done, Failed: server.Name, Err: err}
		}
		if len(done) == 0 {
			*resp = r
		}
		done = append(done, server.Name)
	}
	return nil
}

func (t *MultiTransport) read(ctx context.Context, req CommandRequest, resp *CommandResponse) error {
	first, err := t.Servers[0].sendRead(ctx, req)
	if err != nil {
		return err
	}
	*resp = first

	report, _ := ctx.Value(driftReporterKey{}).(func(Drift))
	for _, server := range t.Servers[1:] {
		r, err := server.sendRead(ctx, req)
		if err != nil {
			return fmt.Errorf("%s: %w", server.Name, err)
		}
		same, err := sameAnswer(first, r)
		if err != nil {
			return fmt.Errorf("%s: %w", server.Name, err)
		}
		if same {
			continue
		}
		drift := Drift{Server: server.Name, Command: req.Command}
		if t.Strict {
			return &DriftError{drift}
		}
		if report != nil {
			report(drift)
		}
	}
	return nil
}

func (s Server) sendRead(ctx context.Context, req CommandRequest) (CommandResponse, error) {
	var resp CommandResponse
	err := s.Send(ctx, req, &resp)
	return resp, err
}

// sameAnswer compares result codes and decoded arguments and ignores text.
func sameAnswer(a, b CommandResponse) (bool, error) {
	if a.Result != b.Result {
		return false, nil
	}
	argsA, err := decodeArguments(a.Arguments)
	if err != nil {
		return false, err
	}
	argsB, err := decodeArguments(b.Arguments)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(argsA, argsB), nil
}

func decodeArguments(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("decoding arguments: %w", err)
	}
	return v, nil
}
