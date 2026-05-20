package agentclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 0},
	}
}

func (c *Client) auth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

type DeployState struct {
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Message string `json:"message"`
	Project string `json:"project"`
	BuildID string `json:"build_id"`
}

func (c *Client) Deploy(ctx context.Context, tenant, buildID, tomlPath, tarballPath string) (DeployState, error) {
	tomlBytes, err := os.ReadFile(tomlPath)
	if err != nil {
		return DeployState{}, err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("tenant", tenant)
	_ = w.WriteField("build_id", buildID)
	_ = w.WriteField("toml", string(tomlBytes))
	tf, err := os.Open(tarballPath)
	if err != nil {
		return DeployState{}, err
	}
	defer tf.Close()
	part, err := w.CreateFormFile("source", "source.tar.gz")
	if err != nil {
		return DeployState{}, err
	}
	if _, err := io.Copy(part, tf); err != nil {
		return DeployState{}, err
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/deploys", &body)
	if err != nil {
		return DeployState{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.auth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return DeployState{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return DeployState{}, fmt.Errorf("deploy: %s", strings.TrimSpace(string(b)))
	}
	var st DeployState
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return DeployState{}, err
	}
	return st, nil
}

func (c *Client) StreamEvents(ctx context.Context, deployID string, fn func(DeployState)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/deploys/"+deployID+"/events", nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("events: %s", strings.TrimSpace(string(b)))
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var st DeployState
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &st); err != nil {
			continue
		}
		fn(st)
		if st.Phase == "succeeded" || st.Phase == "failed" {
			if st.Phase == "failed" {
				return fmt.Errorf("%s", st.Message)
			}
			return nil
		}
	}
	return sc.Err()
}

func (c *Client) Rollback(ctx context.Context, tenant, project string) error {
	url := fmt.Sprintf("%s/projects/%s/%s/rollback", c.baseURL, tenant, project)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rollback: %s", strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) Logs(ctx context.Context, tenant, project, service, tail string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/%s/logs?tail=%s", c.baseURL, tenant, project, tail)
	if service != "" {
		url += "&service=" + service
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("logs: %s", strings.TrimSpace(string(b)))
	}
	return string(b), err
}

func (c *Client) SecretSet(ctx context.Context, tenant, key, value string) error {
	url := fmt.Sprintf("%s/secrets/%s?tenant=%s", c.baseURL, key, tenant)
	body, _ := json.Marshal(map[string]string{"value": value})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("secret set: %s", strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) SecretList(ctx context.Context, tenant string) ([]string, error) {
	url := fmt.Sprintf("%s/secrets?tenant=%s", c.baseURL, tenant)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Secrets []string `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Secrets, nil
}

func (c *Client) DBCreate(ctx context.Context, tenant, name, engine, version string) error {
	body, _ := json.Marshal(map[string]string{
		"tenant":  tenant,
		"name":    name,
		"engine":  engine,
		"version": version,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/databases", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("db create: %s", strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) DBList(ctx context.Context, tenant string) ([]string, error) {
	url := fmt.Sprintf("%s/databases?tenant=%s", c.baseURL, tenant)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Databases []struct {
			Name string `json:"name"`
		} `json:"databases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	names := make([]string, len(out.Databases))
	for i, d := range out.Databases {
		names[i] = d.Name
	}
	return names, nil
}

func (c *Client) DBURL(ctx context.Context, tenant, name string) (string, error) {
	url := fmt.Sprintf("%s/databases/%s/url?tenant=%s", c.baseURL, name, tenant)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.URL, nil
}

func (c *Client) WaitHealthy(ctx context.Context) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
		if err != nil {
			return err
		}
		resp, err := c.http.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	return fmt.Errorf("agent not reachable at %s (is hangar-agent running? is the ssh tunnel up?)", c.baseURL)
}
