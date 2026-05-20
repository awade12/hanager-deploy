package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct{}

func New() *Client {
	return &Client{}
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("docker %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) Available(ctx context.Context) error {
	_, err := c.run(ctx, "version", "--format", "{{.Server.Version}}")
	return err
}
