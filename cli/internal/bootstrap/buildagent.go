package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/awade12/hanager-deploy/cli/internal/version"
)

func BuildAgentForPlatform(ctx context.Context, plat Platform, explicit string) (path string, cleanup func(), err error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", nil, err
		}
		return explicit, func() {}, nil
	}
	tmp, err := os.CreateTemp("", "hangar-agent-linux-*")
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()

	mod := version.Module() + "/agent/cmd/hangar-agent@v0.2.3"
	fmt.Printf("building hangar-agent for %s/%s ...\n", plat.GOOS, plat.GOARCH)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", tmpPath, mod)
	cmd.Env = append(os.Environ(),
		"GOOS="+plat.GOOS,
		"GOARCH="+plat.GOARCH,
		"CGO_ENABLED=0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("cross-build hangar-agent: %w", err)
	}
	return tmpPath, func() { os.Remove(tmpPath) }, nil
}

func ResolveAgentForVPS(ctx context.Context, cfg config.Config, explicit string) (path string, cleanup func(), err error) {
	if _, err := exec.LookPath("go"); err != nil && explicit == "" {
		return "", nil, fmt.Errorf("go not found; install Go to cross-build hangar-agent for the VPS")
	}
	plat, err := DetectPlatform(ctx, cfg)
	if err != nil {
		return "", nil, err
	}
	return BuildAgentForPlatform(ctx, plat, explicit)
}
