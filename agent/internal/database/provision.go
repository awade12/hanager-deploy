package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/awade12/hanager-deploy/agent/internal/docker"
)

type Provisioner struct {
	docker *docker.Client
}

func NewProvisioner(d *docker.Client) *Provisioner {
	return &Provisioner{docker: d}
}

type CreateOpts struct {
	Tenant  string
	Name    string
	Engine  string
	Version string
	Network string
}

func (p *Provisioner) Create(ctx context.Context, opts CreateOpts) (Record, error) {
	switch opts.Engine {
	case "postgres":
		return p.postgres(ctx, opts)
	case "redis":
		return p.redis(ctx, opts)
	default:
		return Record{}, fmt.Errorf("unsupported engine %q", opts.Engine)
	}
}

func (p *Provisioner) postgres(ctx context.Context, opts CreateOpts) (Record, error) {
	pass, err := randomPass()
	if err != nil {
		return Record{}, err
	}
	user := "hangar"
	dbName := "hangar"
	container := dbContainerName(opts.Tenant, opts.Name)
	tag := "postgres:16"
	if opts.Version != "" {
		tag = "postgres:" + opts.Version
	}
	_, err = p.docker.Run(ctx, docker.RunOpts{
		Image:   tag,
		Name:    container,
		Network: opts.Network,
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + pass,
			"POSTGRES_DB=" + dbName,
		},
		Detach: true,
	})
	if err != nil {
		return Record{}, err
	}
	host := container
	port := 5432
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, pass, host, port, dbName)
	return Record{
		Tenant:    opts.Tenant,
		Name:      opts.Name,
		Engine:    "postgres",
		Version:   opts.Version,
		Network:   opts.Network,
		Container: container,
		URL:       url,
		Host:      host,
		Port:      port,
	}, nil
}

func (p *Provisioner) redis(ctx context.Context, opts CreateOpts) (Record, error) {
	container := dbContainerName(opts.Tenant, opts.Name)
	tag := "redis:7"
	if opts.Version != "" {
		tag = "redis:" + opts.Version
	}
	_, err := p.docker.Run(ctx, docker.RunOpts{
		Image:   tag,
		Name:    container,
		Network: opts.Network,
		Detach:  true,
	})
	if err != nil {
		return Record{}, err
	}
	host := container
	port := 6379
	url := fmt.Sprintf("redis://%s:%d", host, port)
	return Record{
		Tenant:    opts.Tenant,
		Name:      opts.Name,
		Engine:    "redis",
		Version:   opts.Version,
		Network:   opts.Network,
		Container: container,
		URL:       url,
		Host:      host,
		Port:      port,
	}, nil
}

func (p *Provisioner) Destroy(ctx context.Context, rec Record) error {
	_ = p.docker.Stop(ctx, rec.Container)
	return p.docker.Remove(ctx, rec.Container)
}

func dbContainerName(tenant, name string) string {
	return fmt.Sprintf("hangar-db-%s-%s", docker.Sanitize(tenant), docker.Sanitize(name))
}

func TenantNetwork(tenant string) string {
	return fmt.Sprintf("hangar-tenant-%s", docker.Sanitize(tenant))
}

func randomPass() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
