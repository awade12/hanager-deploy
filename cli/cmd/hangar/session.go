package main

import (
	"context"
	"fmt"

	"github.com/awade12/hanager-deploy/cli/internal/agentclient"
	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/awade12/hanager-deploy/cli/internal/tunnel"
)

func withAgent(ctx context.Context, fn func(*agentclient.Client, config.Config) error) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	tun, err := tunnel.Start(cfg)
	if err != nil {
		return err
	}
	defer tun.Close()
	client := agentclient.New(cfg.AgentURL(), cfg.Token)
	if err := client.WaitHealthy(ctx); err != nil {
		return fmt.Errorf(`%w

hangar-agent on the VPS is not reachable (connection refused on port 8741).
  hangar doctor
  ssh -i %s %s@%s 'sudo systemctl restart hangar-agent && curl -sf http://127.0.0.1:8741/health'`, err, cfg.KeyPath, cfg.User, cfg.Host)
	}
	return fn(client, cfg)
}
