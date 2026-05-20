package deploy

import (
	"context"
	"fmt"
)

func (p *Pipeline) probeURL(ctx context.Context, container, url string, timeoutSec int) error {
	if timeoutSec < 1 {
		timeoutSec = 2
	}
	py := fmt.Sprintf(
		"import urllib.request; urllib.request.urlopen(%q, timeout=%d)",
		url, timeoutSec,
	)
	if err := p.docker.Exec(ctx, container, []string{"python3", "-c", py}); err == nil {
		return nil
	}
	if err := p.docker.Exec(ctx, container, []string{
		"curl", "-sf", "--max-time", fmt.Sprintf("%d", timeoutSec), url,
	}); err == nil {
		return nil
	}
	if err := p.docker.Exec(ctx, container, []string{
		"wget", "-qO-", fmt.Sprintf("--timeout=%d", timeoutSec), url,
	}); err == nil {
		return nil
	}
	return fmt.Errorf("probe %s failed", url)
}
