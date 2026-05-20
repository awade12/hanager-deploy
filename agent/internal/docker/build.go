package docker

import (
	"context"
	"fmt"
	"path/filepath"
)

type BuildOpts struct {
	Context    string
	Dockerfile string
	Tag        string
}

func ImageTag(tenant, project, service, buildID string) string {
	return fmt.Sprintf("hangar/%s/%s/%s:%s",
		Sanitize(tenant), Sanitize(project), Sanitize(service), Sanitize(buildID))
}

func (c *Client) Build(ctx context.Context, opts BuildOpts) error {
	df := opts.Dockerfile
	if df == "" {
		df = "Dockerfile"
	}
	dfPath := filepath.Join(opts.Context, df)
	args := []string{
		"build",
		"-t", opts.Tag,
		"-f", dfPath,
		opts.Context,
	}
	_, err := c.run(ctx, args...)
	return err
}
