package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func Example() Config {
	cfg := Default()
	cfg.Host = "your-vps.example.com"
	cfg.User = "ubuntu"
	cfg.KeyPath = "~/.ssh/id_rsa"
	return cfg
}
