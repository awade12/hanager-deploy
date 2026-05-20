package tunnel

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/awade12/hanager-deploy/cli/internal/config"
)

type Tunnel struct {
	cmd    *exec.Cmd
	shared bool
}

func Start(cfg config.Config) (*Tunnel, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host not set in ~/.hangar/config.json")
	}
	port := cfg.LocalPort
	if port == 0 {
		port = 8741
	}
	if agentHealthy(port) {
		return &Tunnel{shared: true}, nil
	}
	if PortInUse(port) {
		_ = killSSHOnPort(port)
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 && agentHealthy(port) {
			return &Tunnel{shared: true}, nil
		}
		if attempt > 0 {
			_ = killSSHOnPort(port)
		}
		t, err := startOnce(cfg)
		if err == nil {
			if waitAgent(port, 5*time.Second) {
				return t, nil
			}
			_ = t.Close()
			last = fmt.Errorf("agent not reachable on 127.0.0.1:%d after tunnel", port)
		} else {
			last = err
		}
		if agentHealthy(port) {
			if t != nil {
				_ = t.Close()
			}
			return &Tunnel{shared: true}, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("ssh tunnel after 3 attempts: %w", last)
}

func agentHealthy(port int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", port), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitAgent(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if agentHealthy(port) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
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
	localPort := cfg.LocalPort
	if localPort == 0 {
		localPort = 8741
	}
	bind := fmt.Sprintf("%d:%s", localPort, cfg.RemoteAgent)
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
		"-o", "LogLevel=ERROR",
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
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ssh exited: %w", err)
		}
	case <-time.After(500 * time.Millisecond):
	}
	return &Tunnel{cmd: cmd}, nil
}

func killSSHOnPort(port int) error {
	if _, err := exec.LookPath("lsof"); err != nil {
		return err
	}
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (t *Tunnel) Close() error {
	if t == nil || t.shared {
		return nil
	}
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	return t.cmd.Process.Kill()
}

func PortInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return true
	}
	_ = ln.Close()
	return false
}
