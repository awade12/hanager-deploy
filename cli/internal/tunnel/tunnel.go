package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/awade12/hanager-deploy/cli/internal/config"
)

type Tunnel struct {
	cmd *exec.Cmd
}

func Start(cfg config.Config) (*Tunnel, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host not set in ~/.hangar/config.json")
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		t, err := startOnce(cfg)
		if err == nil {
			time.Sleep(400 * time.Millisecond)
			return t, nil
		}
		last = err
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("ssh tunnel after 3 attempts: %w", last)
}

func startOnce(cfg config.Config) (*Tunnel, error) {
	user := cfg.User
	if user == "" {
		user = "root"
	}
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	bind := fmt.Sprintf("%d:%s", cfg.LocalPort, cfg.RemoteAgent)
	target := fmt.Sprintf("%s@%s", user, cfg.Host)
	args := []string{
		"-F", "/dev/null",
		"-N",
		"-L", bind,
		"-p", strconv.Itoa(port),
		"-o", "ExitOnForwardFailure=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
	}
	if cfg.KeyPath != "" {
		args = append(args, "-i", cfg.KeyPath, "-o", "IdentitiesOnly=yes")
	}
	args = append(args, target)
	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	return &Tunnel{cmd: cmd}, nil
}

func (t *Tunnel) Close() error {
	if t == nil || t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	return t.cmd.Process.Kill()
}
