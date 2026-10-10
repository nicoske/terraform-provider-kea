package keadhcp4

import (
	"context"

	"github.com/wfdewith/terraform-provider-kea/kea"
)

func (c *Client) GetClientClasses(ctx context.Context) ([]string, error) {
	result, err := kea.ExecWithResponse[struct {
		ClientClasses []struct {
			Name string `json:"name"`
		} `json:"client-classes"`
	}](ctx, c.transport, "class-list", struct{}{})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return []string{}, nil
	}

	names := make([]string, 0, len(result.ClientClasses))
	for _, class := range result.ClientClasses {
		names = append(names, class.Name)
	}
	return names, nil
}

func (c *Client) GetClientClass(ctx context.Context, name string) (*ClientClass, error) {
	result, err := kea.ExecWithResponse[struct {
		ClientClasses []ClientClass `json:"client-classes"`
	}](ctx, c.transport, "class-get", struct {
		Name string `json:"name"`
	}{name})

	if err != nil {
		return nil, err
	}

	if result == nil || len(result.ClientClasses) == 0 {
		return nil, nil
	}

	return &result.ClientClasses[0], nil
}

func (c *Client) AddClientClass(ctx context.Context, class ClientClass) error {
	return c.exec(ctx, "client class added", "class-add", struct {
		ClientClasses []ClientClass `json:"client-classes"`
	}{[]ClientClass{class}})
}

// UpdateClientClass replaces the client class with the same name, and adds it
// to the servers that do not hold it.
func (c *Client) UpdateClientClass(ctx context.Context, class ClientClass) error {
	arguments := struct {
		ClientClasses []ClientClass `json:"client-classes"`
	}{[]ClientClass{class}}
	return c.mutate(ctx, "client class updated", func() error {
		return c.upsert(ctx, "class-add", "class-update", arguments, func(server *Client) (bool, error) {
			found, err := server.GetClientClass(ctx, class.Name)
			return found != nil, err
		})
	})
}

func (c *Client) DeleteClientClass(ctx context.Context, name string) error {
	return c.exec(ctx, "client class deleted", "class-del", struct {
		Name string `json:"name"`
	}{name})
}
