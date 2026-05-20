package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/awade12/hanager-deploy/pkg/duration"
	"github.com/awade12/hanager-deploy/pkg/schema"
)

func (p *Pipeline) waitHealthy(ctx context.Context, manifest *schema.Manifest, containers []ContainerRecord) error {
	byService := groupByService(containers)
	for _, svc := range manifest.Services {
		list := byService[svc.Name]
		if len(list) == 0 {
			continue
		}
		if svc.Healthcheck == nil {
		if err := p.waitReachable(ctx, svc, list); err != nil {
			return err
		}
			continue
		}
		if err := p.waitHTTPHealth(ctx, svc, list); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) waitReachable(ctx context.Context, svc schema.Service, containers []ContainerRecord) error {
	port := servicePort(svc)
	deadline := time.Now().Add(60 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		allOK := true
		for _, c := range containers {
			if err := p.probeURL(ctx, c.Name, url, 2); err != nil {
				allOK = false
				break
			}
		}
		if allOK {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("service %s did not become reachable", svc.Name)
}

func (p *Pipeline) waitHTTPHealth(ctx context.Context, svc schema.Service, containers []ContainerRecord) error {
	hc := svc.Healthcheck
	interval, err := duration.Parse(hc.Interval, 10*time.Second)
	if err != nil {
		return err
	}
	timeout, err := duration.Parse(hc.Timeout, 2*time.Second)
	if err != nil {
		return err
	}
	grace, err := duration.Parse(hc.GracePeriod, 30*time.Second)
	if err != nil {
		return err
	}
	path := hc.Path
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	port := servicePort(svc)
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	timeoutSec := int(timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 2
	}

	deadline := time.Now().Add(grace + interval*10)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		allOK := true
		for _, c := range containers {
			if err := p.probeURL(ctx, c.Name, url, timeoutSec); err != nil {
				allOK = false
				break
			}
		}
		if allOK {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("healthcheck failed for service %s (%s)", svc.Name, path)
}

func groupByService(containers []ContainerRecord) map[string][]ContainerRecord {
	out := make(map[string][]ContainerRecord)
	for _, c := range containers {
		out[c.Service] = append(out[c.Service], c)
	}
	return out
}

func servicePort(svc schema.Service) int {
	if svc.Port > 0 {
		return svc.Port
	}
	return 8080
}
