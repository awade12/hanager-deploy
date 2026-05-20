package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	adminURL   string
	httpClient *http.Client
}

func New(adminURL string) *Client {
	return &Client{
		adminURL: strings.TrimRight(adminURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) EnsureServer(ctx context.Context, mode EdgeMode) error {
	cfg := map[string]any{
		"listen": mode.ListenAddrs(),
		"routes": []any{},
	}
	if mode.Public {
		cfg["automatic_https"] = map[string]any{}
	} else {
		cfg["automatic_https"] = map[string]any{"disable": true}
	}
	body, _ := json.Marshal(cfg)
	path := c.adminURL + "/config/apps/http/servers/srv0"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.do(req); err != nil {
		return c.postServer(ctx, mode)
	}
	return nil
}

func (c *Client) postServer(ctx context.Context, mode EdgeMode) error {
	cfg := map[string]any{
		"listen": mode.ListenAddrs(),
		"routes": []any{},
	}
	if mode.Public {
		cfg["automatic_https"] = map[string]any{}
	} else {
		cfg["automatic_https"] = map[string]any{"disable": true}
	}
	body, _ := json.Marshal(cfg)
	path := c.adminURL + "/config/apps/http/servers/srv0"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) deleteID(ctx context.Context, id string) error {
	path := fmt.Sprintf("%s/id/%s", c.adminURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	_ = c.do(req)
	return nil
}

func (c *Client) postJSON(ctx context.Context, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) patchJSON(ctx context.Context, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL+"/config/", nil)
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy %s %s: %s", req.Method, req.URL.Path, strings.TrimSpace(string(b)))
	}
	return nil
}
