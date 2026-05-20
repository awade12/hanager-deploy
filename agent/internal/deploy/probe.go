package deploy

import (
	"context"
	"fmt"
)

func (p *Pipeline) probeURL(ctx context.Context, container, url string, timeoutSec int) error {
	if timeoutSec < 1 {
		timeoutSec = 2
	}
	err := p.docker.Exec(ctx, container, []string{
		"wget", "-qO-", fmt.Sprintf("--timeout=%d", timeoutSec), url,
	})
	if err == nil {
		return nil
	}
	script := fmt.Sprintf(
		`python3 -c "import urllib.request; urllib.request.urlopen(%q, timeout=%d)"`,
		url, timeoutSec,
	)
	if err2 := p.docker.Exec(ctx, container, []string{"sh", "-c", script}); err2 == nil {
		return nil
	}
	err3 := p.docker.Exec(ctx, container, []string{
		"curl", "-sf", "--max-time", fmt.Sprintf("%d", timeoutSec), url,
	})
	if err3 == nil {
		return nil
	}
	return fmt.Errorf("probe %s: %w", url, err)
}
