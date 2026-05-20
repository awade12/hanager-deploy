package bootstrap

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/awade12/hanager-deploy/cli/internal/config"
)

func sshOutput(ctx context.Context, cfg config.Config, remoteCmd string) (string, error) {
	args := sshArgs(cfg, remoteCmd)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("ssh: %s", msg)
		}
		return "", err
	}
	return string(out), nil
}

func DetectPlatform(ctx context.Context, cfg config.Config) (Platform, error) {
	out, err := sshOutput(ctx, cfg, "uname -s && uname -m")
	if err != nil {
		return Platform{GOOS: "linux", GOARCH: "amd64"}, nil
	}
	lines := strings.Fields(strings.TrimSpace(out))
	if len(lines) < 2 {
		return Platform{GOOS: "linux", GOARCH: "amd64"}, nil
	}
	osName := strings.ToLower(lines[0])
	arch := strings.ToLower(lines[1])
	goos := "linux"
	if strings.Contains(osName, "linux") {
		goos = "linux"
	}
	goarch := "amd64"
	switch arch {
	case "x86_64", "amd64":
		goarch = "amd64"
	case "aarch64", "arm64":
		goarch = "arm64"
	case "i386", "i686":
		goarch = "386"
	}
	return Platform{GOOS: goos, GOARCH: goarch}, nil
}

type Platform struct {
	GOOS   string
	GOARCH string
}

func scpPortArgs(cfg config.Config) []string {
	if cfg.Port != 0 && cfg.Port != 22 {
		return []string{"-P", strconv.Itoa(cfg.Port)}
	}
	return nil
}
