package docker

import (
	"context"
	"fmt"
	"strings"
)

func ProjectNetwork(tenant, project string) string {
	return fmt.Sprintf("hangar-%s-%s", Sanitize(tenant), Sanitize(project))
}

func EdgeNetwork() string {
	return "hangar-edge"
}

func (c *Client) EnsureNetwork(ctx context.Context, name string) error {
	_, err := c.run(ctx, "network", "inspect", name)
	if err == nil {
		return nil
	}
	_, err = c.run(ctx, "network", "create", name)
	return err
}

func (c *Client) ConnectNetwork(ctx context.Context, network, container string) error {
	_, err := c.run(ctx, "network", "connect", network, container)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}
