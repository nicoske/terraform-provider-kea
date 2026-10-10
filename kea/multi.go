package kea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
// *PartialWriteError. A read goes to every server too and returns the answer
// of the first server that answers. A server that answers differently, or
// that cannot be reached, is reported to the function set with
// WithDriftReporter, or fails the read with a *DriftError when Strict is set.
// A read fails when no server answers. A single server is used as is.
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

// Drift is a server whose answer to a read command differs from the answer
// used, or a server that could not be reached when Err is set.
type Drift struct {
	Server  string
	Command string
	Err     error
}

// Unavailable reports whether the server could not be reached.
func (d Drift) Unavailable() bool {
	return d.Err != nil
}

// DriftError is returned by a MultiTransport with Strict set when a server
// answers a read command differently from the answer used, or cannot be
// reached.
type DriftError struct {
	Drift
}

func (e *DriftError) Error() string {
	if e.Unavailable() {
		return fmt.Sprintf("servers out of sync: %s did not answer %s: %v", e.Server, e.Command, e.Err)
	}
	return fmt.Sprintf("servers out of sync: %s answers differently to %s", e.Server, e.Command)
}

func (e *DriftError) Unwrap() error {
	return e.Err
}

// UnavailableError is returned when no server answers a read command. Errs
// holds the error of each server in Servers.
type UnavailableError struct {
	Command string
	Servers []string
	Errs    []error
}

func (e *UnavailableError) Error() string {
	failures := make([]string, len(e.Servers))
	for i, server := range e.Servers {
		failures[i] = fmt.Sprintf("%s: %v", server, e.Errs[i])
	}
	return fmt.Sprintf("%s failed on every server: %s", e.Command, strings.Join(failures, "; "))
}

func (e *UnavailableError) Unwrap() []error {
	return e.Errs
}

type driftReporterKey struct{}

// WithDriftReporter returns a context that makes a MultiTransport call report
// for every server that answers a read command differently from the answer
// used, or that cannot be reached.
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

// Subset returns a MultiTransport to the servers named in names, in the order
// of Servers, with the same Strict setting.
func (t *MultiTransport) Subset(names ...string) *MultiTransport {
	subset := &MultiTransport{Strict: t.Strict}
	for _, server := range t.Servers {
		if slices.Contains(names, server.Name) {
			subset.Servers = append(subset.Servers, server)
		}
	}
	return subset
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
	report, _ := ctx.Value(driftReporterKey{}).(func(Drift))
	var first *CommandResponse
	var unavailable UnavailableError
	for _, server := range t.Servers {
		r, err := server.sendRead(ctx, req)
		if isTransportError(err) {
			drift := Drift{Server: server.Name, Command: req.Command, Err: err}
			if t.Strict {
				return &DriftError{drift}
			}
			unavailable.Servers = append(unavailable.Servers, server.Name)
			unavailable.Errs = append(unavailable.Errs, err)
			continue
		}
		if err != nil {
			return fmt.Errorf("%s: %w", server.Name, err)
		}
		if first == nil {
			first = &r
			continue
		}
		same, err := sameAnswer(*first, r)
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

	if first == nil {
		unavailable.Command = req.Command
		return &unavailable
	}
	*resp = *first
	if report != nil {
		for i, server := range unavailable.Servers {
			report(Drift{Server: server, Command: req.Command, Err: unavailable.Errs[i]})
		}
	}
	return nil
}

// isTransportError reports whether err means that the server could not be
// reached, as opposed to an answer that could not be used.
func isTransportError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
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
