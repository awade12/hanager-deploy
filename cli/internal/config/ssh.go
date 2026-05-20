package config

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandKeyPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}

func (c *Config) ResolvedKeyPath() (string, error) {
	return ExpandKeyPath(c.KeyPath)
}
