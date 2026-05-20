package database

import (
	"os"
	"path/filepath"

	"github.com/awade12/hanager-deploy/pkg/fsutil"
)

type Record struct {
	Tenant    string `json:"tenant"`
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	Version   string `json:"version"`
	Network   string `json:"network"`
	Container string `json:"container"`
	URL       string `json:"url"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
}

type Catalog struct {
	root string
}

func NewCatalog(dataDir string) *Catalog {
	return &Catalog{root: filepath.Join(dataDir, "databases")}
}

func (c *Catalog) path(tenant string) string {
	return filepath.Join(c.root, sanitize(tenant)+".json")
}

type tenantDBs struct {
	Databases map[string]Record `json:"databases"`
}

func (c *Catalog) Load(tenant string) (map[string]Record, error) {
	path := c.path(tenant)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return map[string]Record{}, nil
	}
	var t tenantDBs
	if err := fsutil.ReadJSON(path, &t); err != nil {
		return nil, err
	}
	if t.Databases == nil {
		return map[string]Record{}, nil
	}
	return t.Databases, nil
}

func (c *Catalog) Save(tenant string, dbs map[string]Record) error {
	if err := os.MkdirAll(c.root, 0o755); err != nil {
		return err
	}
	return fsutil.WriteJSONAtomic(c.path(tenant), tenantDBs{Databases: dbs})
}

func sanitize(t string) string {
	out := make([]byte, 0, len(t))
	for i := 0; i < len(t); i++ {
		c := t[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
