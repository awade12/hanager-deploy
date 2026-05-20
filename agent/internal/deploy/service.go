package deploy

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/hangar-sh/hangar/pkg/schema"
)

type CreateInput struct {
	Tenant  string
	BuildID string
	TOML    []byte
	Tarball []byte
}

type Service struct {
	store    *Store
	pipeline *Pipeline
	rollback *Rollback
	logs     *Logs
}

func NewService(store *Store, pipeline *Pipeline, rollback *Rollback, logs *Logs) *Service {
	return &Service{
		store:    store,
		pipeline: pipeline,
		rollback: rollback,
		logs:     logs,
	}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (State, error) {
	if len(in.Tarball) == 0 {
		return State{}, fmt.Errorf("source tarball required")
	}
	manifest, err := schema.Parse(in.TOML)
	if err != nil {
		return State{}, fmt.Errorf("invalid hangar.toml: %w", err)
	}
	id := uuid.NewString()
	st := State{
		ID:      id,
		Tenant:  in.Tenant,
		Project: manifest.Project.Name,
		BuildID: in.BuildID,
		Phase:   PhasePending,
	}
	if err := s.store.Ensure(id); err != nil {
		return State{}, err
	}
	if err := os.WriteFile(s.store.TOMLPath(id), in.TOML, 0o644); err != nil {
		return State{}, err
	}
	if err := os.WriteFile(s.store.TarballPath(id), in.Tarball, 0o644); err != nil {
		return State{}, err
	}
	if err := s.store.WriteState(st); err != nil {
		return State{}, err
	}
	go s.pipeline.Run(context.WithoutCancel(ctx), id)
	return st, nil
}

func (s *Service) Get(id string) (State, error) {
	return s.store.ReadState(id)
}

func (s *Service) Rollback(ctx context.Context, tenant, project string) error {
	return s.rollback.Run(ctx, tenant, project)
}

func (s *Service) Logs(ctx context.Context, w io.Writer, tenant, project, service string, follow bool, tail string) error {
	return s.logs.Write(ctx, w, tenant, project, service, follow, tail)
}
