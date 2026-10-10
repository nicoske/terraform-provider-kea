package keadhcp4

import (
	"context"
	"net/netip"

	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func (c *Client) GetReservations(ctx context.Context, target kea.OperationTarget, subnetID uint32) ([]Reservation, error) {
	result, err := kea.ExecWithResponse[struct {
		Hosts []Reservation `json:"hosts"`
	}](ctx, c.transport, "reservation-get-all", kea.WithTarget(target, struct {
		SubnetID uint32 `json:"subnet-id"`
	}{subnetID}))

	if err != nil {
		return nil, err
	}

	if result == nil {
		return []Reservation{}, nil
	}

	return result.Hosts, nil
}

func (c *Client) GetReservation(ctx context.Context, target kea.OperationTarget, query keaquery.ReservationQuery) (*Reservation, error) {
	return kea.ExecWithResponse[Reservation](ctx, c.transport, "reservation-get", kea.WithTarget(target, query))
}

func (c *Client) AddReservation(ctx context.Context, target kea.OperationTarget, reservation Reservation) error {
	return c.exec(ctx, "reservation added", "reservation-add", kea.WithTarget(target, struct {
		Reservation Reservation `json:"reservation"`
	}{reservation}))
}

// UpdateReservation replaces the reservation with the same subnet and
// identifier, and adds it to the servers that do not hold it.
func (c *Client) UpdateReservation(ctx context.Context, target kea.OperationTarget, reservation Reservation) error {
	arguments := kea.WithTarget(target, struct {
		Reservation Reservation `json:"reservation"`
	}{reservation})
	return c.mutate(ctx, "reservation updated", func() error {
		return c.upsert(ctx, "reservation-add", "reservation-update", arguments, func(server *Client) (bool, error) {
			found, err := server.GetReservation(ctx, target, reservation.query())
			return found != nil, err
		})
	})
}

func (c *Client) DeleteReservation(ctx context.Context, target kea.OperationTarget, query keaquery.ReservationQuery) error {
	return c.exec(ctx, "reservation deleted", "reservation-del", kea.WithTarget(target, query))
}

// query identifies the reservation by the first identifier it holds, or by
// its address when it has none.
func (r Reservation) query() keaquery.ReservationQuery {
	for _, id := range []struct {
		kind  string
		value kea.HexID
	}{
		{"circuit-id", r.CircuitID},
		{"client-id", r.ClientID},
		{"duid", r.DUID},
		{"flex-id", r.FlexID},
		{"hw-address", r.HWAddress},
	} {
		if len(id.value) > 0 {
			return keaquery.ReservationByIdentifier(r.SubnetID, id.kind, id.value.String())
		}
	}
	var ip netip.Addr
	if r.IPAddress != nil {
		ip = *r.IPAddress
	}
	return keaquery.ReservationByIP(r.SubnetID, ip)
}
