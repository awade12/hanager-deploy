package caddy

import (
	"context"
	"fmt"
)

func RouteID(tenant, project, service string) string {
	return fmt.Sprintf("hangar-%s-%s-%s",
		sanitizePart(tenant), sanitizePart(project), sanitizePart(service))
}

func DefaultHost(project, service string) string {
	return fmt.Sprintf("%s-%s.hangar.local", project, service)
}

type RouteOpts struct {
	RouteID   string
	Hosts     []string
	Upstreams []string
}

func (c *Client) UpsertRoute(ctx context.Context, opts RouteOpts) error {
	if err := c.patchUpstreamsByID(ctx, opts.RouteID, opts.Upstreams); err == nil {
		return nil
	}
	return c.postRoute(ctx, opts)
}

func (c *Client) patchUpstreamsByID(ctx context.Context, routeID string, dials []string) error {
	upstreams := make([]map[string]string, len(dials))
	for i, d := range dials {
		upstreams[i] = map[string]string{"dial": d}
	}
	path := fmt.Sprintf("%s/id/%s/handle/0/upstreams", c.adminURL, routeID)
	return c.patchJSON(ctx, path, upstreams)
}

func (c *Client) postRoute(ctx context.Context, opts RouteOpts) error {
	hosts := opts.Hosts
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts for route %s", opts.RouteID)
	}
	upstreams := make([]map[string]string, len(opts.Upstreams))
	for i, d := range opts.Upstreams {
		upstreams[i] = map[string]string{"dial": d}
	}
	route := map[string]any{
		"@id":   opts.RouteID,
		"match": []map[string]any{{"host": hosts}},
		"handle": []map[string]any{{
			"handler":   "reverse_proxy",
			"upstreams": upstreams,
		}},
	}
	_ = c.deleteID(ctx, opts.RouteID)
	path := c.adminURL + "/config/apps/http/servers/srv0/routes"
	return c.postJSON(ctx, path, route)
}

func UpstreamDial(containerName string, port int) string {
	return fmt.Sprintf("%s:%d", containerName, port)
}

func sanitizePart(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			out = append(out, c)
		} else {
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "x"
	}
	return string(out)
}
