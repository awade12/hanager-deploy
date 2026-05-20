package bootstrap

import (
	"context"
	"strings"

	"github.com/awade12/hanager-deploy/cli/internal/config"
)

func SSHRun(ctx context.Context, cfg config.Config, remoteCmd, stdin string) error {
	return sshRun(ctx, cfg, remoteCmd, stdin)
}

func SSHWriteFile(ctx context.Context, cfg config.Config, remotePath, content string) error {
	return sshRun(ctx, cfg, "sudo tee "+remotePath+" > /dev/null", content)
}

func SSHHost(cfg config.Config) string {
	return sshTarget(cfg)
}

func SSHKeyPath(cfg config.Config) string {
	return cfg.KeyPath
}

func SSHOutput(ctx context.Context, cfg config.Config, remoteCmd string) (string, error) {
	return sshOutput(ctx, cfg, remoteCmd)
}

func TrimSSHOut(s string) string {
	return strings.TrimSpace(s)
}
