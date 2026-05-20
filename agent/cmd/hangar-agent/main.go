package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/awade12/hanager-deploy/agent/internal/api"
	"github.com/awade12/hanager-deploy/agent/internal/caddy"
	"github.com/awade12/hanager-deploy/agent/internal/config"
	"github.com/awade12/hanager-deploy/agent/internal/database"
	"github.com/awade12/hanager-deploy/agent/internal/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/docker"
	"github.com/awade12/hanager-deploy/agent/internal/runtime"
	"github.com/awade12/hanager-deploy/agent/internal/secret"
)

func main() {
	configPath := flag.String("config", "/etc/hangar/agent.json", "agent config path")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	deployRoot := config.DeploysDir(cfg.DataDir)
	if err := os.MkdirAll(deployRoot, 0o755); err != nil {
		logger.Error("mkdir deploys", "err", err)
		os.Exit(1)
	}

	dockerClient := docker.New()
	if err := waitDocker(context.Background(), dockerClient, 60*time.Second, logger); err != nil {
		logger.Error("docker unavailable", "err", err)
		os.Exit(1)
	}

	secretStore, err := secret.Open(cfg.DataDir)
	if err != nil {
		logger.Error("secrets", "err", err)
		os.Exit(1)
	}

	dbSvc := database.NewService(cfg.DataDir, dockerClient)
	envResolver := deploy.NewEnvResolver(secretStore, dbSvc)

	store := deploy.NewStore(deployRoot)
	if err := deploy.NewRecoverer(store, dockerClient, logger).Run(context.Background()); err != nil {
		logger.Error("crash recovery", "err", err)
		os.Exit(1)
	}

	caddyClient := caddy.New(cfg.CaddyAdminURL)
	caddyDir := filepath.Join(cfg.DataDir, "caddy")
	edge := caddy.NewEnsurer(dockerClient, caddyClient, cfg.CaddyContainer, cfg.CaddyHTTPPort, caddyDir)
	rt := runtime.NewStore(config.RuntimeDir(cfg.DataDir))
	pipeline := deploy.NewPipeline(store, rt, dockerClient, caddyClient, edge, envResolver, dbSvc, logger)
	rollback := deploy.NewRollback(rt, dockerClient, caddyClient)
	logs := deploy.NewLogs(rt, dockerClient)
	deploySvc := deploy.NewService(store, pipeline, rollback, logs)
	srv := api.New(deploySvc, rt, secretStore, dbSvc, cfg.Token)

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("agent listening", "addr", cfg.ListenAddr, "data_dir", cfg.DataDir)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func waitDocker(ctx context.Context, d *docker.Client, timeout time.Duration, logger *slog.Logger) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := d.Available(ctx); err == nil {
			return nil
		}
		logger.Info("waiting for docker")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return d.Available(ctx)
}
