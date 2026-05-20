package deploy

import (
	"context"
	"fmt"

	"github.com/hangar-sh/hangar/agent/internal/caddy"
	"github.com/hangar-sh/hangar/agent/internal/docker"
	"github.com/hangar-sh/hangar/agent/internal/runtime"
)

type Rollback struct {
	runtime *runtime.Store
	docker  *docker.Client
	caddy   *caddy.Client
}

func NewRollback(rt *runtime.Store, d *docker.Client, c *caddy.Client) *Rollback {
	return &Rollback{runtime: rt, docker: d, caddy: c}
}

func (r *Rollback) Run(ctx context.Context, tenant, project string) error {
	cur, err := r.runtime.Load(tenant, project)
	if err != nil {
		return err
	}
	if cur.Previous == nil {
		return fmt.Errorf("no previous deployment to roll back to")
	}
	prev := *cur.Previous
	if prev.BuildID == "" && len(prev.Services) == 0 {
		return fmt.Errorf("no previous deployment to roll back to")
	}

	for _, svc := range cur.Services {
		for _, c := range svc.Containers {
			_ = r.docker.Stop(ctx, c.Name)
		}
	}

	for _, svc := range prev.Services {
		if !svc.Public {
			continue
		}
		dials := make([]string, 0, len(svc.Containers))
		for _, c := range svc.Containers {
			_ = r.docker.Start(ctx, c.Name)
			dials = append(dials, caddy.UpstreamDial(c.Name, svc.Port))
		}
		if len(dials) == 0 {
			continue
		}
		routeID := svc.CaddyRouteID
		if routeID == "" {
			routeID = caddy.RouteID(tenant, project, svc.Name)
		}
		hosts := svc.Hosts
		if len(hosts) == 0 {
			hosts = []string{caddy.DefaultHost(project, svc.Name)}
		}
		if err := r.caddy.UpsertRoute(ctx, caddy.RouteOpts{
			RouteID:   routeID,
			Hosts:     hosts,
			Upstreams: dials,
		}); err != nil {
			return fmt.Errorf("caddy rollback %s: %w", svc.Name, err)
		}
	}

	restored := prev
	restored.Tenant = tenant
	restored.Project = project
	restored.Previous = nil
	restored.PreviousBuildID = cur.BuildID
	return r.runtime.Save(restored)
}
