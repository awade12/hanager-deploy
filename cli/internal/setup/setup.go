package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/awade12/hanager-deploy/cli/internal/bootstrap"
	"github.com/awade12/hanager-deploy/cli/internal/config"
)

type Options struct {
	KeyPath string
	DataDir string
}

func Run(ctx context.Context, target Target, opts Options) error {
	keyPath := opts.KeyPath
	if keyPath == "" {
		var err error
		keyPath, err = DefaultKeyPath()
		if err != nil {
			return err
		}
	}
	expanded, err := config.ExpandKeyPath(keyPath)
	if err != nil {
		return err
	}
	keyPath = expanded

	if err := EnsureKey(keyPath); err != nil {
		return err
	}

	fmt.Printf("==> connecting to %s\n", target.SSHAddr())
	if err := ensureSSH(target, keyPath); err != nil {
		return err
	}

	cfg := config.Default()
	cfg.Host = target.Host
	cfg.User = target.User
	cfg.Port = target.Port
	cfg.KeyPath = keyPath
	if err := config.Save(cfg); err != nil {
		return err
	}
	path, _ := config.Path()
	fmt.Printf("==> wrote %s\n", path)

	fmt.Println("==> installing hangar agent on VPS")
	agentBin, err := bootstrap.ResolveAgentBinary(ctx, "")
	if err != nil {
		return err
	}
	if err := bootstrap.Run(ctx, cfg, bootstrap.Options{
		AgentBinary: agentBin,
		DataDir:     opts.DataDir,
	}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Setup complete.")
	fmt.Println("  cd your-app   # directory with hangar.toml")
	fmt.Println("  hangar deploy")
	return nil
}

func ensureSSH(target Target, keyPath string) error {
	if err := config.TestSSHPort(target.User, target.Host, target.Port, keyPath); err == nil {
		fmt.Println("SSH OK")
		return nil
	}
	if _, err := exec.LookPath("ssh-copy-id"); err == nil {
		fmt.Println("==> copying SSH key to server (enter VPS password if asked)")
		if err := runSSHCopyID(target, keyPath); err == nil {
			if err := config.TestSSHPort(target.User, target.Host, target.Port, keyPath); err == nil {
				fmt.Println("SSH OK")
				return nil
			}
		}
	}
	return manualSSHHelp(target, keyPath)
}

func runSSHCopyID(target Target, keyPath string) error {
	args := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if target.Port != 0 && target.Port != 22 {
		args = append(args, "-p", strconv.Itoa(target.Port))
	}
	args = append(args, target.SSHAddr())
	cmd := exec.Command("ssh-copy-id", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func manualSSHHelp(target Target, keyPath string) error {
	pub, err := PublicKeyLine(keyPath)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Could not connect with SSH yet. Add this public key on the VPS:")
	fmt.Println()
	fmt.Println(pub)
	fmt.Println("On the server (provider console or existing login), run:")
	fmt.Println()
	fmt.Printf("  mkdir -p ~/.ssh && chmod 700 ~/.ssh\n")
	fmt.Printf("  echo '%s' >> ~/.ssh/authorized_keys\n", trimLine(pub))
	fmt.Printf("  chmod 600 ~/.ssh/authorized_keys\n")
	fmt.Println()
	fmt.Println("Then run again:")
	fmt.Printf("  hangar setup %s\n", target.SSHAddr())
	return fmt.Errorf("ssh not configured for %s", target.SSHAddr())
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
