package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "check VPS agent and SSH connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			user := cfg.User
			if user == "" {
				user = "ubuntu"
			}
			target := user + "@" + cfg.Host
			fmt.Printf("==> SSH %s\n", target)
			if err := runSSH(cfg, target, "echo ok"); err != nil {
				return fmt.Errorf("ssh: %w", err)
			}
			fmt.Println("SSH OK")

			fmt.Println("==> hangar-agent service")
			check := `systemctl is-active hangar-agent 2>/dev/null || echo inactive; curl -sf http://127.0.0.1:8741/health 2>/dev/null || echo health_failed`
			out, err := runSSHOutput(cfg, target, check)
			if err != nil {
				return err
			}
			fmt.Println(out)
			if strings.Contains(out, "inactive") || strings.Contains(out, "failed") || strings.Contains(out, "health_failed") {
				fmt.Println()
				fmt.Println("Agent is not healthy. On the VPS run:")
				fmt.Printf("  ssh -i %s %s\n", cfg.KeyPath, target)
				fmt.Println("  sudo systemctl restart hangar-agent")
				fmt.Println("  sudo systemctl status hangar-agent")
				fmt.Println("  curl http://127.0.0.1:8741/health")
				fmt.Println()
				fmt.Println("Or from your Mac after SSH works:")
				fmt.Println("  hangar init")
				return fmt.Errorf("hangar-agent not running on VPS")
			}
			fmt.Println("Agent OK")
			return nil
		},
	}
}

func runSSH(cfg config.Config, target, remote string) error {
	_, err := runSSHOutput(cfg, target, remote)
	return err
}

func runSSHOutput(cfg config.Config, target, remote string) (string, error) {
	args := []string{"-F", "/dev/null", "-o", "BatchMode=yes", "-o", "ConnectTimeout=15", "-o", "StrictHostKeyChecking=accept-new"}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath, "-o", "IdentitiesOnly=yes")
	}
	if cfg.Port != 0 && cfg.Port != 22 {
		args = append(args, "-p", strconv.Itoa(cfg.Port))
	}
	args = append(args, target, remote)
	cmd := exec.Command("ssh", args...)
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	return string(b), err
}
