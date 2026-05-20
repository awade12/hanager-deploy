package database

import (
	"context"
	"fmt"

	"github.com/awade12/hanager-deploy/agent/internal/docker"
	"github.com/awade12/hanager-deploy/pkg/schema"
)

type Service struct {
	catalog     *Catalog
	provisioner *Provisioner
	docker      *docker.Client
}

func NewService(dataDir string, d *docker.Client) *Service {
	return &Service{
		catalog:     NewCatalog(dataDir),
		provisioner: NewProvisioner(d),
		docker:      d,
	}
}

func (s *Service) Create(ctx context.Context, tenant, name, engine, version string) (Record, error) {
	dbs, err := s.catalog.Load(tenant)
	if err != nil {
		return Record{}, err
	}
	if _, exists := dbs[name]; exists {
		return Record{}, fmt.Errorf("database %q already exists", name)
	}
	net := TenantNetwork(tenant)
	if err := s.docker.EnsureNetwork(ctx, net); err != nil {
		return Record{}, err
	}
	rec, err := s.provisioner.Create(ctx, CreateOpts{
		Tenant:  tenant,
		Name:    name,
		Engine:  engine,
		Version: version,
		Network: net,
	})
	if err != nil {
		return Record{}, err
	}
	dbs[name] = rec
	if err := s.catalog.Save(tenant, dbs); err != nil {
		return Record{}, err
	}
	return rec, nil
}

func (s *Service) List(tenant string) ([]Record, error) {
	dbs, err := s.catalog.Load(tenant)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(dbs))
	for _, r := range dbs {
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) URL(tenant, name string) (string, error) {
	dbs, err := s.catalog.Load(tenant)
	if err != nil {
		return "", err
	}
	r, ok := dbs[name]
	if !ok {
		return "", fmt.Errorf("database %q not found", name)
	}
	return r.URL, nil
}

func (s *Service) Destroy(ctx context.Context, tenant, name string) error {
	dbs, err := s.catalog.Load(tenant)
	if err != nil {
		return err
	}
	r, ok := dbs[name]
	if !ok {
		return fmt.Errorf("database %q not found", name)
	}
	if err := s.provisioner.Destroy(ctx, r); err != nil {
		return err
	}
	delete(dbs, name)
	return s.catalog.Save(tenant, dbs)
}

func (s *Service) EnsureProjectDBs(ctx context.Context, tenant, project, network string, dbs []schema.Database) error {
	tenantDBs, err := s.catalog.Load(tenant)
	if err != nil {
		return err
	}
	changed := false
	for _, spec := range dbs {
		if _, ok := tenantDBs[spec.Name]; ok {
			if err := s.docker.ConnectNetwork(ctx, network, tenantDBs[spec.Name].Container); err != nil {
				return err
			}
			continue
		}
		rec, err := s.provisioner.Create(ctx, CreateOpts{
			Tenant:  tenant,
			Name:    spec.Name,
			Engine:  spec.Engine,
			Version: spec.Version,
			Network: network,
		})
		if err != nil {
			return fmt.Errorf("database %s: %w", spec.Name, err)
		}
		tenantDBs[spec.Name] = rec
		changed = true
	}
	if changed {
		return s.catalog.Save(tenant, tenantDBs)
	}
	return nil
}

func (s *Service) ResolveURL(tenant, project, ref string, tenantScoped bool) (string, error) {
	dbs, err := s.catalog.Load(tenant)
	if err != nil {
		return "", err
	}
	name := ref
	if tenantScoped {
		r, ok := dbs[name]
		if !ok {
			return "", fmt.Errorf("tenant database %q not found", name)
		}
		return r.URL, nil
	}
	r, ok := dbs[name]
	if !ok {
		return "", fmt.Errorf("database %q not found (create via hangar db or [[database]] in hangar.toml)", name)
	}
	return r.URL, nil
}
