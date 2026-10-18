package keadhcp4_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
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

func configGet(dhcp4 string) kea.CommandResponse {
	return kea.CommandResponse{Result: kea.ResultSuccess, Arguments: json.RawMessage(`{"Dhcp4":` + dhcp4 + `,"hash":"x"}`)}
}

func failing(command string) kea.CommandResponse {
	return kea.CommandResponse{Result: kea.ResultError, Text: command + " failed"}
}

var testQuery = keaquery.ReservationByIdentifier(1, "hw-address", "02:00:00:00:00:01")

func deleteReservation(client *keadhcp4.Client) error {
	return client.DeleteReservation(context.Background(), kea.OperationTargetMemory, testQuery)
}

func TestClient_configWriteAfterChange(t *testing.T) {
	server := &fakeServer{}

	if err := deleteReservation(keadhcp4.NewClient(server)); err != nil {
		t.Fatal(err)
	}
	if want := []string{"reservation-del", "config-write"}; !slices.Equal(server.commands, want) {
		t.Fatalf("commands: %v, expected %v", server.commands, want)
	}
}

func TestClient_noConfigWriteAfterFailedChange(t *testing.T) {
	server := &fakeServer{answers: map[string]kea.CommandResponse{"reservation-del": failing("reservation-del")}}

	err := deleteReservation(keadhcp4.NewClient(server))
	var writeErr *keadhcp4.ConfigWriteError
	if err == nil || errors.As(err, &writeErr) {
		t.Fatalf("expected the reservation-del error, got %v", err)
	}
	if want := []string{"reservation-del"}; !slices.Equal(server.commands, want) {
		t.Fatalf("commands: %v, expected %v", server.commands, want)
	}
}

func TestClient_configWriteError(t *testing.T) {
	server := &fakeServer{answers: map[string]kea.CommandResponse{"config-write": failing("config-write")}}

	err := deleteReservation(keadhcp4.NewClient(server))
	var writeErr *keadhcp4.ConfigWriteError
	if !errors.As(err, &writeErr) {
		t.Fatalf("expected ConfigWriteError, got %v", err)
	}
	if want := "reservation deleted but config-write failed: general error: config-write failed"; err.Error() != want {
		t.Fatalf("error: %q, expected %q", err.Error(), want)
	}
}

func TestClient_noConfigWriteForDatabaseReservation(t *testing.T) {
	server := &fakeServer{}
	if err := keadhcp4.NewClient(server).DeleteReservation(context.Background(), kea.OperationTargetDatabase, testQuery); err != nil {
		t.Fatal(err)
	}
	if want := []string{"reservation-del"}; !slices.Equal(server.commands, want) {
		t.Fatalf("commands: %v, expected %v", server.commands, want)
	}
}

func TestClient_reservationTarget(t *testing.T) {
	cases := map[string]struct {
		dhcp4 string
		want  kea.OperationTarget
	}{
		"hosts-database":        {`{"hosts-database":{"type":"postgresql"}}`, kea.OperationTargetDatabase},
		"hosts-databases":       {`{"hosts-databases":[{"type":"postgresql"}]}`, kea.OperationTargetDatabase},
		"empty hosts-databases": {`{"hosts-databases":[]}`, kea.OperationTargetMemory},
		"no hosts database":     {`{"lease-database":{"type":"memfile"}}`, kea.OperationTargetMemory},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := &fakeServer{answers: map[string]kea.CommandResponse{"config-get": configGet(tc.dhcp4)}}
			client := keadhcp4.NewClient(server)

			for range 2 {
				target, err := client.ReservationTarget(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if target != tc.want {
					t.Fatalf("target: %s, expected %s", target, tc.want)
				}
			}
			if want := []string{"config-get"}; !slices.Equal(server.commands, want) {
				t.Fatalf("commands: %v, expected %v", server.commands, want)
			}
		})
	}
}

func TestClient_reservationTargetErrorNotCached(t *testing.T) {
	server := &fakeServer{answers: map[string]kea.CommandResponse{"config-get": failing("config-get")}}
	client := keadhcp4.NewClient(server)

	if _, err := client.ReservationTarget(context.Background()); err == nil {
		t.Fatal("expected an error")
	}

	server.answers["config-get"] = configGet(`{}`)
	target, err := client.ReservationTarget(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if target != kea.OperationTargetMemory {
		t.Fatalf("target: %s, expected memory", target)
	}
}
