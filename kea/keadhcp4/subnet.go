package keadhcp4

import (
	"context"

	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func (c *Client) GetSubnets(ctx context.Context, target kea.OperationTarget) ([]Subnet, error) {
	result, err := kea.ExecWithResponse[struct {
		Subnets []Subnet `json:"subnets"`
	}](ctx, c.transport, "subnet4-list", kea.WithTarget(target, struct{}{}))

	if err != nil {
		return nil, err
	}

	if result == nil {
		return []Subnet{}, nil
	}

	return result.Subnets, nil
}

func (c *Client) GetSubnet(ctx context.Context, query keaquery.SubnetQuery) (*Subnet, error) {
	result, err := kea.ExecWithResponse[struct {
		Subnet4 []Subnet `json:"subnet4"`
	}](ctx, c.transport, "subnet4-get", query)

	if err != nil {
		return nil, err
	}

	if result == nil || len(result.Subnet4) == 0 {
		return nil, nil
	}

	return &result.Subnet4[0], nil
}

func (c *Client) AddSubnet(ctx context.Context, subnet Subnet) error {
	err := kea.Exec(ctx, c.transport, "subnet4-add", struct {
		Subnet4 []Subnet `json:"subnet4"`
	}{[]Subnet{subnet}})
	if err != nil {
		return err
	}
	return c.persist(ctx, "subnet added")
}

// UpdateSubnet replaces the subnet with the same ID, and adds it to the
// servers that do not hold it.
func (c *Client) UpdateSubnet(ctx context.Context, subnet Subnet) error {
	arguments := struct {
		Subnet4 []Subnet `json:"subnet4"`
	}{[]Subnet{subnet}}
	err := c.upsert(ctx, "subnet4-add", "subnet4-update", arguments, func(server *Client) (bool, error) {
		found, err := server.GetSubnet(ctx, keaquery.SubnetByID(subnet.ID))
		return found != nil, err
	})
	if err != nil {
		return err
	}
	return c.persist(ctx, "subnet updated")
}

func (c *Client) DeleteSubnet(ctx context.Context, id uint32) error {
	err := kea.Exec(ctx, c.transport, "subnet4-del", struct {
		ID uint32 `json:"id"`
	}{id})
	if err != nil {
		return err
	}
	return c.persist(ctx, "subnet deleted")
}
