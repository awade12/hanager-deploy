package main

import (
	"context"

	"github.com/hangar-sh/hangar/cli/internal/agentclient"
	"github.com/hangar-sh/hangar/cli/internal/config"
	"github.com/hangar-sh/hangar/cli/internal/tunnel"
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
		return err
	}
	return fn(client, cfg)
}
