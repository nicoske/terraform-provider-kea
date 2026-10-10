package keadhcp4

import (
	"context"

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
	err := kea.Exec(ctx, c.transport, "reservation-add", kea.WithTarget(target, struct {
		Reservation Reservation `json:"reservation"`
	}{reservation}))
	if err != nil {
		return err
	}
	return c.persist(ctx, "reservation added")
}

func (c *Client) UpdateReservation(ctx context.Context, target kea.OperationTarget, reservation Reservation) error {
	err := kea.Exec(ctx, c.transport, "reservation-update", kea.WithTarget(target, struct {
		Reservation Reservation `json:"reservation"`
	}{reservation}))
	if err != nil {
		return err
	}
	return c.persist(ctx, "reservation updated")
}

func (c *Client) DeleteReservation(ctx context.Context, target kea.OperationTarget, query keaquery.ReservationQuery) error {
	if err := kea.Exec(ctx, c.transport, "reservation-del", kea.WithTarget(target, query)); err != nil {
		return err
	}
	return c.persist(ctx, "reservation deleted")
}
