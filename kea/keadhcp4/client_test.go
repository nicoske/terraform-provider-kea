package keadhcp4_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"testing"

	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

// fakeServer answers success to every command except those in answers.
type fakeServer struct {
	answers  map[string]kea.CommandResponse
	commands []string
}

func (f *fakeServer) Send(ctx context.Context, req kea.CommandRequest, resp *kea.CommandResponse) error {
	f.commands = append(f.commands, req.Command)
	if answer, ok := f.answers[req.Command]; ok {
		*resp = answer
		return nil
	}
	*resp = kea.CommandResponse{Result: kea.ResultSuccess}
	return nil
}

func servers(fakes ...*fakeServer) *kea.MultiTransport {
	t := &kea.MultiTransport{}
	for i, f := range fakes {
		t.Servers = append(t.Servers, kea.Server{Name: string(rune('a' + i)), Transport: f})
	}
	return t
}

func holding(command, arguments string) *fakeServer {
	return &fakeServer{answers: map[string]kea.CommandResponse{
		command: {Result: kea.ResultSuccess, Arguments: json.RawMessage(arguments)},
	}}
}

func missing(command string) *fakeServer {
	return &fakeServer{answers: map[string]kea.CommandResponse{
		command: {Result: kea.ResultEmpty, Text: "not found"},
	}}
}

func TestClient_updateAddsWhereMissing(t *testing.T) {
	ip := netip.MustParseAddr("10.0.0.10")
	reservation := keadhcp4.Reservation{SubnetID: 1, HWAddress: kea.HexID{0x02, 0, 0, 0, 0, 0x01}, IPAddress: &ip}

	for name, tc := range map[string]struct {
		get, add, update string
		call             func(*keadhcp4.Client) error
	}{
		"subnet": {"subnet4-get", "subnet4-add", "subnet4-update", func(c *keadhcp4.Client) error {
			return c.UpdateSubnet(context.Background(), keadhcp4.Subnet{ID: 1, Subnet: netip.MustParsePrefix("10.0.0.0/24")})
		}},
		"client class": {"class-get", "class-add", "class-update", func(c *keadhcp4.Client) error {
			return c.UpdateClientClass(context.Background(), keadhcp4.ClientClass{Name: "x"})
		}},
		"reservation": {"reservation-get", "reservation-add", "reservation-update", func(c *keadhcp4.Client) error {
			return c.UpdateReservation(context.Background(), kea.OperationTargetMemory, reservation)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			arguments := map[string]string{
				"subnet4-get":     `{"subnet4": [{"id": 1, "subnet": "10.0.0.0/24"}]}`,
				"class-get":       `{"client-classes": [{"name": "x"}]}`,
				"reservation-get": `{"subnet-id": 1, "hw-address": "02:00:00:00:00:01"}`,
			}[tc.get]
			a, b := holding(tc.get, arguments), missing(tc.get)

			if err := tc.call(keadhcp4.NewClient(servers(a, b))); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(a.commands, []string{tc.get, tc.update}) {
				t.Fatalf("commands sent to a: %v", a.commands)
			}
			if !slices.Equal(b.commands, []string{tc.get, tc.add}) {
				t.Fatalf("commands sent to b: %v", b.commands)
			}
		})
	}
}

func TestClient_updateStopsAtFirstFailure(t *testing.T) {
	a := holding("subnet4-get", `{"subnet4": [{"id": 1, "subnet": "10.0.0.0/24"}]}`)
	b := missing("subnet4-get")
	b.answers["subnet4-add"] = kea.CommandResponse{Result: kea.ResultError, Text: "subnet with the prefix of '10.0.0.0/24' already exists"}
	c := missing("subnet4-get")

	err := keadhcp4.NewClient(servers(a, b, c)).UpdateSubnet(context.Background(), keadhcp4.Subnet{ID: 1, Subnet: netip.MustParsePrefix("10.0.0.0/24")})

	var partial *kea.PartialWriteError
	if !errors.As(err, &partial) {
		t.Fatalf("expected PartialWriteError, got %v", err)
	}
	want := "subnet4-update succeeded on a and failed on b: general error: subnet with the prefix of '10.0.0.0/24' already exists"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
	if len(c.commands) != 0 {
		t.Fatalf("commands sent after failure: %v", c.commands)
	}
}
