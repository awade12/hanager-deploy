package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hangar-sh/hangar/agent/internal/docker"
	"github.com/hangar-sh/hangar/agent/internal/runtime"
)

type Logs struct {
	runtime *runtime.Store
	docker  *docker.Client
}

func NewLogs(rt *runtime.Store, d *docker.Client) *Logs {
	return &Logs{runtime: rt, docker: d}
}

func (l *Logs) Write(ctx context.Context, w io.Writer, tenant, project, service string, follow bool, tail string) error {
	rt, err := l.runtime.Load(tenant, project)
	if err != nil {
		return err
	}
	if service == "" {
		for name, svc := range rt.Services {
			for _, c := range svc.Containers {
				if err := l.writeContainer(ctx, w, name, c.Name, follow, tail); err != nil {
					return err
				}
			}
		}
		return nil
	}
	svc, ok := rt.Services[service]
	if !ok {
		return fmt.Errorf("unknown service %q", service)
	}
	for _, c := range svc.Containers {
		if err := l.writeContainer(ctx, w, service, c.Name, follow, tail); err != nil {
			return err
		}
	}
	return nil
}

func (l *Logs) writeContainer(ctx context.Context, w io.Writer, service, name string, follow bool, tail string) error {
	if tail == "" {
		tail = "100"
	}
	out, err := l.docker.Logs(ctx, name, follow, tail)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "=== %s (%s) ===\n", service, name)
	_, err = io.WriteString(w, strings.TrimSpace(out)+"\n")
	return err
}
