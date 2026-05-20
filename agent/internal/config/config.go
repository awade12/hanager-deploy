package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddr     string `json:"listen_addr"`
	DataDir        string `json:"data_dir"`
	Token          string `json:"token"`
	CaddyAdminURL  string `json:"caddy_admin_url"`
	CaddyHTTPPort  int    `json:"caddy_http_port"`
	CaddyContainer string `json:"caddy_container"`
	PublicEdge     bool   `json:"public_edge"`
	ACMEEmail      string `json:"acme_email"`
}

func Default() Config {
	return Config{
		ListenAddr:     "127.0.0.1:8741",
		DataDir:        "/var/lib/hangar",
		CaddyAdminURL:  "http://127.0.0.1:2019",
		CaddyHTTPPort:  8877,
		CaddyContainer: "hangar-caddy",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.DataDir == "" {
		cfg.DataDir = Default().DataDir
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = Default().ListenAddr
	}
	if cfg.CaddyAdminURL == "" {
		cfg.CaddyAdminURL = Default().CaddyAdminURL
	}
	if cfg.CaddyHTTPPort == 0 {
		cfg.CaddyHTTPPort = Default().CaddyHTTPPort
	}
	if cfg.CaddyContainer == "" {
		cfg.CaddyContainer = Default().CaddyContainer
	}
	return cfg, nil
}

func DeploysDir(dataDir string) string {
	return filepath.Join(dataDir, "deploys")
}

func RuntimeDir(dataDir string) string {
	return filepath.Join(dataDir, "runtime")
}
