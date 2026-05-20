package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Host        string `json:"host"`
	User        string `json:"user"`
	Port        int    `json:"port"`
	KeyPath     string `json:"key_path"`
	Token       string `json:"token"`
	Tenant      string `json:"tenant"`
	LocalPort   int    `json:"local_port"`
	RemoteAgent string `json:"remote_agent"`
}

func Default() Config {
	return Config{
		User:        "root",
		Port:        22,
		Tenant:      "default",
		LocalPort:   8741,
		RemoteAgent: "127.0.0.1:8741",
	}
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hangar", "config.json"), nil
}

func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, fmt.Errorf("config not found at %s", path)
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.LocalPort == 0 {
		cfg.LocalPort = Default().LocalPort
	}
	if cfg.RemoteAgent == "" {
		cfg.RemoteAgent = Default().RemoteAgent
	}
	if cfg.Tenant == "" {
		cfg.Tenant = Default().Tenant
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.KeyPath != "" {
		expanded, err := ExpandKeyPath(cfg.KeyPath)
		if err != nil {
			return cfg, err
		}
		cfg.KeyPath = expanded
	}
	return cfg, nil
}

func (c Config) AgentURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.LocalPort)
}
