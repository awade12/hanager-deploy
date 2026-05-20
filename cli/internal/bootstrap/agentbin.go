package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/awade12/hanager-deploy/cli/internal/version"
)

func ResolveAgentBinary(ctx context.Context, explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("agent binary %s: %w", explicit, err)
		}
		return explicit, nil
	}
	if p, err := exec.LookPath("hangar-agent"); err == nil {
		return p, nil
	}
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("hangar-agent not on PATH; install with: go install %s/agent/cmd/hangar-agent@latest", version.Module())
	}
	install := version.Module() + "/agent/cmd/hangar-agent@v0.1.0"
	cmd := exec.CommandContext(ctx, "go", "install", install)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go install hangar-agent: %w", err)
	}
	p := filepath.Join(goBinDir(), agentBinaryName())
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("hangar-agent not found at %s after go install", p)
	}
	return p, nil
}

func goBinDir() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "bin"
		}
		gopath = filepath.Join(home, "go")
	}
	return filepath.Join(gopath, "bin")
}

func agentBinaryName() string {
	if runtime.GOOS == "windows" {
		return "hangar-agent.exe"
	}
	return "hangar-agent"
}
