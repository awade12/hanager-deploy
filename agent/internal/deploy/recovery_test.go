package deploy_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/awade12/hanager-deploy/agent/internal/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/docker"
)

func TestRecoverBuildingMarksFailed(t *testing.T) {
	dir := t.TempDir()
	store := deploy.NewStore(dir)
	d := docker.New()
	id := "d1"
	if err := store.Ensure(id); err != nil {
		t.Fatal(err)
	}
	st := deploy.State{ID: id, Phase: deploy.PhaseBuilding, BuildID: "b1"}
	if err := store.WriteState(st); err != nil {
		t.Fatal(err)
	}
	if err := deploy.NewRecoverer(store, d, slog.Default()).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadState(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != deploy.PhaseFailed {
		t.Fatalf("phase = %s", got.Phase)
	}
}

func TestRecoverSwappedMarksFailed(t *testing.T) {
	dir := t.TempDir()
	store := deploy.NewStore(dir)
	d := docker.New()
	id := "d2"
	if err := store.Ensure(id); err != nil {
		t.Fatal(err)
	}
	st := deploy.State{ID: id, Phase: deploy.PhaseSwapped, BuildID: "b2"}
	if err := store.WriteState(st); err != nil {
		t.Fatal(err)
	}
	if err := deploy.NewRecoverer(store, d, slog.Default()).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := store.ReadState(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != deploy.PhaseFailed {
		t.Fatalf("phase = %s", got.Phase)
	}
}
