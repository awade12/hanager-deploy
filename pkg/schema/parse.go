package schema

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("toml syntax: %w", err)
	}
	if err := Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}
