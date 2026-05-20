package caddy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awade12/hanager-deploy/agent/internal/docker"
)

type Ensurer struct {
	docker    *docker.Client
	caddy     *Client
	container string
	httpPort  int
	configDir string
}

func NewEnsurer(d *docker.Client, c *Client, container string, httpPort int, configDir string) *Ensurer {
	return &Ensurer{
		docker:    d,
		caddy:     c,
		container: container,
		httpPort:  httpPort,
		configDir: configDir,
	}
}

func (e *Ensurer) Ensure(ctx context.Context) error {
	if err := e.docker.EnsureNetwork(ctx, docker.EdgeNetwork()); err != nil {
		return fmt.Errorf("edge network: %w", err)
	}
	if err := e.ensureCaddyContainer(ctx); err != nil {
		return err
	}
	if err := e.waitAdmin(ctx); err != nil {
		return err
	}
	listen := fmt.Sprintf(":%d", e.httpPort)
	return e.caddy.EnsureServer(ctx, listen)
}

func (e *Ensurer) waitAdmin(ctx context.Context) error {
	var last error
	for i := 0; i < 30; i++ {
		if err := e.caddy.Ping(ctx); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("caddy admin: %w", last)
}

func (e *Ensurer) ensureCaddyContainer(ctx context.Context) error {
	running, err := e.docker.IsRunning(ctx, e.container)
	if err == nil && running {
		return e.docker.ConnectNetwork(ctx, docker.EdgeNetwork(), e.container)
	}
	_ = e.docker.Remove(ctx, e.container)

	if err := os.MkdirAll(e.configDir, 0o755); err != nil {
		return err
	}
	caddyfile := filepath.Join(e.configDir, "Caddyfile")
	content := `{
	admin 0.0.0.0:2019
}
`
	if err := os.WriteFile(caddyfile, []byte(content), 0o644); err != nil {
		return err
	}

	caddyfileAbs, err := filepath.Abs(caddyfile)
	if err != nil {
		return err
	}
	_, err = e.docker.RunDetached(ctx, docker.RunDetachedOpts{
		Name:    e.container,
		Image:   "caddy:2.8-alpine",
		Network: docker.EdgeNetwork(),
		Ports: []string{
			"127.0.0.1:2019:2019",
			fmt.Sprintf("127.0.0.1:%d:%d", e.httpPort, e.httpPort),
		},
		Mounts:  []string{fmt.Sprintf("%s:/etc/caddy/Caddyfile:ro", caddyfileAbs)},
		Command: []string{"caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"},
	})
	return err
}

func (e *Ensurer) ConnectProject(ctx context.Context, network string) error {
	return e.docker.ConnectNetwork(ctx, network, e.container)
}
