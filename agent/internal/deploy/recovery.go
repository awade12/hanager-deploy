package deploy

import (
	"context"
	"log/slog"

	"github.com/awade12/hanager-deploy/agent/internal/docker"
)

type Recoverer struct {
	store  *Store
	docker *docker.Client
	logger *slog.Logger
}

func NewRecoverer(store *Store, d *docker.Client, logger *slog.Logger) *Recoverer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Recoverer{store: store, docker: d, logger: logger}
}

func (r *Recoverer) Run(ctx context.Context) error {
	states, err := r.store.ListAll()
	if err != nil {
		return err
	}
	for _, st := range states {
		if st.IsTerminal() {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := r.recoverOne(ctx, st); err != nil {
			r.logger.Error("deploy recovery failed", "id", st.ID, "err", err)
		}
	}
	return nil
}

func (r *Recoverer) recoverOne(ctx context.Context, st State) error {
	switch st.Phase {
	case PhaseBuilding:
		st.Phase = PhaseFailed
		st.Message = "agent restarted during build"
		return r.store.WriteState(st)
	case PhaseContainersStarted, PhaseHealthcheck:
		r.cleanup(ctx, st.NewContainers)
		st.Phase = PhaseFailed
		st.Message = "agent restarted before caddy swap"
		return r.store.WriteState(st)
	case PhaseSwapped, PhaseDraining:
		r.drain(ctx, st.DrainContainers)
		r.cleanup(ctx, st.NewContainers)
		st.Phase = PhaseFailed
		st.Message = "agent restarted mid-drain; manual verify required"
		return r.store.WriteState(st)
	case PhasePending:
		return nil
	default:
		return nil
	}
}

func (r *Recoverer) cleanup(ctx context.Context, containers []ContainerRecord) {
	for _, c := range containers {
		if err := r.docker.Stop(ctx, c.Name); err != nil {
			r.logger.Warn("recovery stop", "container", c.Name, "err", err)
		}
		_ = r.docker.Remove(ctx, c.Name)
	}
}

func (r *Recoverer) drain(ctx context.Context, targets []ContainerRecord) {
	for _, c := range targets {
		_ = r.docker.Stop(ctx, c.Name)
	}
}
