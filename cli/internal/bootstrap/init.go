package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/awade12/hanager-deploy/cli/internal/config"
)

type Options struct {
	AgentBinary string
	DataDir     string
}

func Run(ctx context.Context, cfg config.Config, opts Options) error {
	if cfg.Host == "" {
		return fmt.Errorf("set host in ~/.hangar/config.json")
	}
	agentPath, cleanup, err := ResolveAgentForVPS(ctx, cfg, opts.AgentBinary)
	if err != nil {
		return err
	}
	defer cleanup()

	if opts.DataDir == "" {
		opts.DataDir = "/var/lib/hangar"
	}
	target := sshTarget(cfg)
	script := buildScript(opts.DataDir)
	if err := sshRun(ctx, cfg, "bash -s", script); err != nil {
		return fmt.Errorf("bootstrap script: %w", err)
	}
	if err := scp(cfg, agentPath, "/tmp/hangar-agent"); err != nil {
		return err
	}
	install := fmt.Sprintf("sudo install -m 755 /tmp/hangar-agent /usr/local/bin/hangar-agent && sudo mkdir -p %s /etc/hangar", opts.DataDir)
	if err := sshRun(ctx, cfg, install, ""); err != nil {
		return err
	}
	agentJSON := fmt.Sprintf(`{
  "listen_addr": "127.0.0.1:8741",
  "data_dir": %q,
  "caddy_http_port": 8877
}
`, opts.DataDir)
	if err := sshRun(ctx, cfg, "sudo tee /etc/hangar/agent.json > /dev/null", agentJSON); err != nil {
		return err
	}
	unit := `[Unit]
Description=hangar agent
After=network-online.target docker.service
Wants=docker.service

[Service]
Type=simple
ExecStart=/usr/local/bin/hangar-agent -config /etc/hangar/agent.json
Restart=on-failure
RestartSec=5
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
`
	if err := sshRun(ctx, cfg, "sudo tee /etc/systemd/system/hangar-agent.service > /dev/null", unit); err != nil {
		return err
	}
	if err := sshRun(ctx, cfg, "sudo systemctl daemon-reload && sudo systemctl enable --now hangar-agent", ""); err != nil {
		return err
	}
	fmt.Printf("hangar agent installed on %s\n", target)
	return nil
}

func buildScript(dataDir string) string {
	return fmt.Sprintf(`set -euo pipefail
if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sudo sh
  sudo usermod -aG docker "$USER" || true
fi
if command -v ufw >/dev/null; then
  sudo ufw --force enable || true
  sudo ufw allow OpenSSH || true
  sudo ufw allow 80/tcp || true
  sudo ufw allow 443/tcp || true
fi
sudo mkdir -p %s
`, dataDir)
}

func sshTarget(cfg config.Config) string {
	user := cfg.User
	if user == "" {
		user = "root"
	}
	return user + "@" + cfg.Host
}

func sshRun(ctx context.Context, cfg config.Config, remoteCmd, stdin string) error {
	args := sshArgs(cfg, remoteCmd)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func scp(cfg config.Config, local, remote string) error {
	args := []string{"-F", "/dev/null", "-o", "PreferredAuthentications=publickey", "-o", "PasswordAuthentication=no"}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath, "-o", "IdentitiesOnly=yes")
	}
	if cfg.Port != 0 {
		args = append(args, "-P", fmt.Sprintf("%d", cfg.Port))
	}
	args = append(args, "-o", "StrictHostKeyChecking=accept-new", local, sshTarget(cfg)+":"+remote)
	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func sshArgs(cfg config.Config, remoteCmd string) []string {
	args := []string{"-F", "/dev/null", "-o", "PreferredAuthentications=publickey", "-o", "PasswordAuthentication=no"}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath, "-o", "IdentitiesOnly=yes")
	}
	if cfg.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", cfg.Port))
	}
	args = append(args, sshTarget(cfg), remoteCmd)
	return args
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o755)
}
