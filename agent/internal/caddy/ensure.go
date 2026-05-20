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
	mode      EdgeMode
	configDir string
}

func NewEnsurer(d *docker.Client, c *Client, container string, mode EdgeMode, configDir string) *Ensurer {
	return &Ensurer{
		docker:    d,
		caddy:     c,
		container: container,
		mode:      mode,
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
	return e.caddy.EnsureServer(ctx, e.mode)
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
	wantMode := "local"
	if e.mode.Public {
		wantMode = "public"
	}
	running, err := e.docker.IsRunning(ctx, e.container)
	if err == nil && running {
		cur, _ := e.docker.ContainerEnv(ctx, e.container, "HANGAR_EDGE")
		if cur == wantMode {
			return e.docker.ConnectNetwork(ctx, docker.EdgeNetwork(), e.container)
		}
	}
	_ = e.docker.Remove(ctx, e.container)

	if err := os.MkdirAll(e.configDir, 0o755); err != nil {
		return err
	}
	if e.mode.Public {
		_ = os.MkdirAll(filepath.Join(e.configDir, "data"), 0o755)
		_ = os.MkdirAll(filepath.Join(e.configDir, "config"), 0o755)
	}
	caddyfile := filepath.Join(e.configDir, "Caddyfile")
	if err := os.WriteFile(caddyfile, []byte(e.mode.CaddyfileGlobal()), 0o644); err != nil {
		return err
	}

	caddyfileAbs, err := filepath.Abs(caddyfile)
	if err != nil {
		return err
	}
	mounts := []string{fmt.Sprintf("%s:/etc/caddy/Caddyfile:ro", caddyfileAbs)}
	mounts = append(mounts, e.mode.ExtraMounts(e.configDir)...)
	_, err = e.docker.RunDetached(ctx, docker.RunDetachedOpts{
		Name:    e.container,
		Image:   "caddy:2.8-alpine",
		Network: docker.EdgeNetwork(),
		Ports:   e.mode.DockerPorts(),
		Mounts:  mounts,
		Env:     []string{"HANGAR_EDGE=" + wantMode},
		Command: []string{"caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"},
	})
	return err
}

func (e *Ensurer) ConnectProject(ctx context.Context, network string) error {
	return e.docker.ConnectNetwork(ctx, network, e.container)
}
