package docker

import (
	"context"
	"fmt"
	"strings"
)

type RunOpts struct {
	Image   string
	Name    string
	Network string
	Command []string
	Env     []string
	Detach  bool
}

type RunOnceOpts struct {
	Image   string
	Network string
	Command []string
	Env     []string
}

type RunDetachedOpts struct {
	Name    string
	Image   string
	Network string
	Ports   []string
	Mounts  []string
	Env     []string
	Command []string
}

func ContainerName(project, service, buildID string, index int) string {
	return fmt.Sprintf("%s-%s-%s-%d",
		Sanitize(project), Sanitize(service), Sanitize(buildID), index)
}

func (c *Client) Run(ctx context.Context, opts RunOpts) (string, error) {
	args := []string{"run"}
	if opts.Detach {
		args = append(args, "-d")
	}
	args = append(args, "--network", opts.Network, "--name", opts.Name)
	for _, e := range opts.Env {
		args = append(args, "-e", e)
	}
	args = append(args, opts.Image)
	args = append(args, opts.Command...)
	return c.run(ctx, args...)
}

func (c *Client) RunOnce(ctx context.Context, opts RunOnceOpts) error {
	args := []string{"run", "--rm", "--network", opts.Network}
	for _, e := range opts.Env {
		args = append(args, "-e", e)
	}
	args = append(args, opts.Image)
	args = append(args, opts.Command...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) RunDetached(ctx context.Context, opts RunDetachedOpts) (string, error) {
	args := []string{"run", "-d", "--name", opts.Name, "--network", opts.Network}
	for _, p := range opts.Ports {
		args = append(args, "-p", p)
	}
	for _, m := range opts.Mounts {
		args = append(args, "-v", m)
	}
	for _, e := range opts.Env {
		args = append(args, "-e", e)
	}
	args = append(args, opts.Image)
	args = append(args, opts.Command...)
	return c.run(ctx, args...)
}

func (c *Client) Stop(ctx context.Context, nameOrID string) error {
	_, err := c.run(ctx, "stop", "-t", "15", nameOrID)
	return err
}

func (c *Client) Start(ctx context.Context, nameOrID string) error {
	_, err := c.run(ctx, "start", nameOrID)
	return err
}

func (c *Client) Remove(ctx context.Context, nameOrID string) error {
	_, err := c.run(ctx, "rm", "-f", nameOrID)
	return err
}

func (c *Client) ContainerEnv(ctx context.Context, name, key string) (string, error) {
	prefix := key + "="
	out, err := c.run(ctx, "inspect", "-f", "{{range .Config.Env}}{{println .}}{{end}}", name)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix), nil
		}
	}
	return "", nil
}

func (c *Client) IsRunning(ctx context.Context, name string) (bool, error) {
	out, err := c.run(ctx, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
}

func (c *Client) Logs(ctx context.Context, name string, follow bool, tail string) (string, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	if tail != "" {
		args = append(args, "--tail", tail)
	}
	args = append(args, name)
	return c.run(ctx, args...)
}

func (c *Client) Exec(ctx context.Context, name string, cmd []string) error {
	args := append([]string{"exec", name}, cmd...)
	_, err := c.run(ctx, args...)
	return err
}
