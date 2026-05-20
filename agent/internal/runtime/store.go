package runtime

import (
	"os"
	"path/filepath"

	"github.com/awade12/hanager-deploy/pkg/fsutil"
)

type Container struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Service struct {
	Name         string      `json:"name"`
	Image        string      `json:"image"`
	Port         int         `json:"port"`
	Public       bool        `json:"public"`
	Hosts        []string    `json:"hosts,omitempty"`
	Containers   []Container `json:"containers"`
	CaddyRouteID string      `json:"caddy_route_id,omitempty"`
}

type Project struct {
	Tenant          string             `json:"tenant"`
	Project         string             `json:"project"`
	BuildID         string             `json:"build_id"`
	PreviousBuildID string             `json:"previous_build_id,omitempty"`
	Services        map[string]Service `json:"services"`
	Previous        *Project           `json:"previous,omitempty"`
}

type Store struct {
	root string
}

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) path(tenant, project string) string {
	return filepath.Join(s.root, sanitize(tenant), sanitize(project)+".json")
}

func (s *Store) Load(tenant, project string) (Project, error) {
	var p Project
	err := fsutil.ReadJSON(s.path(tenant, project), &p)
	if os.IsNotExist(err) {
		return Project{Tenant: tenant, Project: project, Services: map[string]Service{}}, nil
	}
	if p.Services == nil {
		p.Services = map[string]Service{}
	}
	return p, err
}

func (s *Store) Save(p Project) error {
	if err := os.MkdirAll(filepath.Dir(s.path(p.Tenant, p.Project)), 0o755); err != nil {
		return err
	}
	return fsutil.WriteJSONAtomic(s.path(p.Tenant, p.Project), p)
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
