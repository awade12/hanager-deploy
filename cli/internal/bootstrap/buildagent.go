package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/awade12/hanager-deploy/cli/internal/version"
)

const agentModuleVersion = "v0.3.0"

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

	fmt.Printf("building hangar-agent for %s/%s ...\n", plat.GOOS, plat.GOARCH)
	if err := crossBuildAgent(ctx, tmpPath, plat); err != nil {
		os.Remove(tmpPath)
		return "", nil, err
	}
	return tmpPath, func() { os.Remove(tmpPath) }, nil
}

func crossBuildAgent(ctx context.Context, out string, plat Platform) error {
	dir, err := os.MkdirTemp("", "hangar-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module hangar.build\n\ngo 1.22\n"), 0o644); err != nil {
		return err
	}

	pkg := version.Module() + "/agent/cmd/hangar-agent"
	env := append(os.Environ(),
		"GOOS="+plat.GOOS,
		"GOARCH="+plat.GOARCH,
		"CGO_ENABLED=0",
	)

	get := exec.CommandContext(ctx, "go", "get", pkg+"@"+agentModuleVersion)
	get.Dir = dir
	get.Env = env
	get.Stdout = os.Stdout
	get.Stderr = os.Stderr
	if err := get.Run(); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	build := exec.CommandContext(ctx, "go", "build", "-o", out, pkg)
	build.Dir = dir
	build.Env = env
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

func ResolveAgentForVPS(ctx context.Context, cfg config.Config, explicit string) (path string, cleanup func(), err error) {
	if _, err := exec.LookPath("go"); err != nil && explicit == "" {
		return "", nil, fmt.Errorf("go not found; install Go 1.22+ to cross-build hangar-agent for the VPS")
	}
	plat, err := DetectPlatform(ctx, cfg)
	if err != nil {
		return "", nil, err
	}
	return BuildAgentForPlatform(ctx, plat, explicit)
}
