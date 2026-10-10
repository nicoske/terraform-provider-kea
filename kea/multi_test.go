package kea_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

type fakeServer struct {
	answers  map[string]kea.CommandResponse
	err      error
	commands []string
}

func (f *fakeServer) Send(ctx context.Context, req kea.CommandRequest, resp *kea.CommandResponse) error {
	f.commands = append(f.commands, req.Command)
	if f.err != nil {
		return f.err
	}
	*resp = f.answers[req.Command]
	return nil
}

func answer(result kea.ResultCode, text, arguments string) kea.CommandResponse {
	resp := kea.CommandResponse{Result: result, Text: text}
	if arguments != "" {
		resp.Arguments = json.RawMessage(arguments)
	}
	return resp
}

func multi(servers ...*fakeServer) *kea.MultiTransport {
	names := []string{"a", "b", "c"}
	t := &kea.MultiTransport{}
	for i, s := range servers {
		t.Servers = append(t.Servers, kea.Server{Name: names[i], Transport: s})
	}
	return t
}

func ok(command string) *fakeServer {
	return &fakeServer{answers: map[string]kea.CommandResponse{
		command: answer(kea.ResultSuccess, "done", ""),
	}}
}

func TestIsReadCommand(t *testing.T) {
	for command, want := range map[string]bool{
		"reservation-get":             true,
		"reservation-get-all":         true,
		"reservation-get-by-hostname": true,
		"subnet4-list":                true,
		"class-get":                   true,
		"config-get":                  true,
		"config-hash-get":             true,
		"config-test":                 true,
		"list-commands":               true,
		"status-get":                  true,
		"version-get":                 true,
		"reservation-add":             false,
		"reservation-del":             false,
		"class-update":                false,
		"config-write":                false,
		"config-set":                  false,
		"config-reload":               false,
	} {
		if got := kea.IsReadCommand(command); got != want {
			t.Errorf("IsReadCommand(%q) = %v, want %v", command, got, want)
		}
	}
}

func TestMultiTransport_write(t *testing.T) {
	a, b := ok("reservation-add"), ok("reservation-add")

	err := kea.Exec(context.Background(), multi(a, b), "reservation-add", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.commands) != 1 || len(b.commands) != 1 {
		t.Fatalf("commands sent: a %v, b %v", a.commands, b.commands)
	}
}

func TestMultiTransport_partialWrite(t *testing.T) {
	for name, b := range map[string]*fakeServer{
		"transport error": {err: errors.New("connection refused")},
		"kea error": {answers: map[string]kea.CommandResponse{
			"reservation-add": answer(kea.ResultError, "subnet not found", ""),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			a, c := ok("reservation-add"), ok("reservation-add")

			err := kea.Exec(context.Background(), multi(a, b, c), "reservation-add", struct{}{})

			var partial *kea.PartialWriteError
			if !errors.As(err, &partial) {
				t.Fatalf("expected PartialWriteError, got %v", err)
			}
			if len(partial.Done) != 1 || partial.Done[0] != "a" || partial.Failed != "b" {
				t.Fatalf("done %v, failed %q", partial.Done, partial.Failed)
			}
			if partial.Command != "reservation-add" {
				t.Fatalf("command %q", partial.Command)
			}
			if len(c.commands) != 0 {
				t.Fatalf("command sent after failure: %v", c.commands)
			}
		})
	}
}

func TestMultiTransport_writeFailsOnFirstServer(t *testing.T) {
	a := &fakeServer{err: errors.New("connection refused")}
	b := ok("reservation-del")

	err := kea.Exec(context.Background(), multi(a, b), "reservation-del", struct{}{})

	want := "reservation-del failed on a: connection refused"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
	if len(b.commands) != 0 {
		t.Fatalf("command sent after failure: %v", b.commands)
	}
}

func TestPartialWriteError(t *testing.T) {
	err := &kea.PartialWriteError{
		Command: "config-write",
		Done:    []string{"a", "b"},
		Failed:  "c",
		Err:     errors.New("general error: no space left"),
	}

	want := "config-write succeeded on a, b and failed on c: general error: no space left"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func readServers(arguments ...string) []*fakeServer {
	var servers []*fakeServer
	for i, args := range arguments {
		servers = append(servers, &fakeServer{answers: map[string]kea.CommandResponse{
			"reservation-get": answer(kea.ResultSuccess, string(rune('a'+i))+" found", args),
		}})
	}
	return servers
}

func execRead(ctx context.Context, trans kea.Transport) (*map[string]any, error) {
	return kea.ExecWithResponse[map[string]any](ctx, trans, "reservation-get", struct{}{})
}

func TestMultiTransport_readInSync(t *testing.T) {
	servers := readServers(
		`{"ip-address": "10.0.0.1", "hw-address": "02:00:00:00:00:01"}`,
		`{"hw-address": "02:00:00:00:00:01", "ip-address": "10.0.0.1"}`,
	)
	var drifts []kea.Drift
	ctx := kea.WithDriftReporter(context.Background(), func(d kea.Drift) { drifts = append(drifts, d) })

	if _, err := execRead(ctx, multi(servers...)); err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 0 {
		t.Fatalf("unexpected drift: %v", drifts)
	}
	for _, s := range servers {
		if len(s.commands) != 1 {
			t.Fatalf("commands sent: %v", s.commands)
		}
	}
}

func TestMultiTransport_readDrift(t *testing.T) {
	servers := readServers(
		`{"ip-address": "10.0.0.1"}`,
		`{"ip-address": "10.0.0.1"}`,
		`{"ip-address": "10.0.0.2"}`,
	)
	var drifts []kea.Drift
	ctx := kea.WithDriftReporter(context.Background(), func(d kea.Drift) { drifts = append(drifts, d) })

	result, err := execRead(ctx, multi(servers...))
	if err != nil {
		t.Fatal(err)
	}
	if (*result)["ip-address"] != "10.0.0.1" {
		t.Fatalf("expected the first server's answer, got %v", *result)
	}
	if len(drifts) != 1 || drifts[0] != (kea.Drift{Server: "c", Command: "reservation-get"}) {
		t.Fatalf("drifts: %v", drifts)
	}
}

func TestMultiTransport_readMissingOnOneServer(t *testing.T) {
	a := readServers(`{"ip-address": "10.0.0.1"}`)[0]
	b := &fakeServer{answers: map[string]kea.CommandResponse{
		"reservation-get": answer(kea.ResultEmpty, "Host not found.", "{}"),
	}}
	var drifts []kea.Drift
	ctx := kea.WithDriftReporter(context.Background(), func(d kea.Drift) { drifts = append(drifts, d) })

	if _, err := execRead(ctx, multi(a, b)); err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 1 || drifts[0].Server != "b" {
		t.Fatalf("drifts: %v", drifts)
	}
}

func TestMultiTransport_readDriftWithoutReporter(t *testing.T) {
	servers := readServers(`{"ip-address": "10.0.0.1"}`, `{"ip-address": "10.0.0.2"}`)

	if _, err := execRead(context.Background(), multi(servers...)); err != nil {
		t.Fatal(err)
	}
}

func TestMultiTransport_strictRead(t *testing.T) {
	servers := readServers(`{"ip-address": "10.0.0.1"}`, `{"ip-address": "10.0.0.2"}`)
	trans := multi(servers...)
	trans.Strict = true

	_, err := execRead(context.Background(), trans)

	var drift *kea.DriftError
	if !errors.As(err, &drift) {
		t.Fatalf("expected DriftError, got %v", err)
	}
	if drift.Server != "b" || drift.Command != "reservation-get" {
		t.Fatalf("drift: %v", drift.Drift)
	}
	want := "servers out of sync: b answers differently to reservation-get"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestMultiTransport_readError(t *testing.T) {
	a := readServers(`{"ip-address": "10.0.0.1"}`)[0]
	b := &fakeServer{err: errors.New("connection refused")}

	_, err := execRead(context.Background(), multi(a, b))

	want := "b: connection refused"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestMultiTransport_singleServer(t *testing.T) {
	a := &fakeServer{answers: map[string]kea.CommandResponse{
		"reservation-add": answer(kea.ResultError, "duplicate", ""),
	}}

	err := kea.Exec(context.Background(), multi(a), "reservation-add", struct{}{})

	want := "general error: duplicate"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}
