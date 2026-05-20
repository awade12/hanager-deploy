package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hangar-sh/hangar/agent/internal/archive"
	"github.com/hangar-sh/hangar/agent/internal/caddy"
	"github.com/hangar-sh/hangar/agent/internal/docker"
	"github.com/hangar-sh/hangar/agent/internal/runtime"
	"github.com/hangar-sh/hangar/pkg/schema"
)

type Pipeline struct {
	store   *Store
	runtime *runtime.Store
	docker  *docker.Client
	caddy   *caddy.Client
	edge    *caddy.Ensurer
	env     *EnvResolver
	db      interface {
		EnsureProjectDBs(ctx context.Context, tenant, project, network string, dbs []schema.Database) error
	}
	logger *slog.Logger
}

func NewPipeline(store *Store, rt *runtime.Store, d *docker.Client, c *caddy.Client, edge *caddy.Ensurer, env *EnvResolver, db interface {
	EnsureProjectDBs(ctx context.Context, tenant, project, network string, dbs []schema.Database) error
}, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		store:   store,
		runtime: rt,
		docker:  d,
		caddy:   c,
		edge:    edge,
		env:     env,
		db:      db,
		logger:  logger,
	}
}

func (p *Pipeline) Run(ctx context.Context, deployID string) {
	if err := p.run(ctx, deployID); err != nil {
		p.logger.Error("deploy failed", "id", deployID, "err", err)
		_ = p.fail(ctx, deployID, err.Error())
	}
}

func (p *Pipeline) run(ctx context.Context, deployID string) error {
	st, err := p.store.ReadState(deployID)
	if err != nil {
		return err
	}

	tomlBytes, err := os.ReadFile(p.store.TOMLPath(deployID))
	if err != nil {
		return err
	}
	manifest, err := schema.Parse(tomlBytes)
	if err != nil {
		return err
	}

	if err := p.extractWorkspace(deployID); err != nil {
		return err
	}
	workspace := p.store.WorkspacePath(deployID)

	prev, err := p.runtime.Load(st.Tenant, st.Project)
	if err != nil {
		return err
	}
	st.DrainContainers = drainTargets(prev)
	st.PreviousBuildID = prev.BuildID

	if err := p.setPhase(deployID, PhaseBuilding, "building images"); err != nil {
		return err
	}

	if err := p.edge.Ensure(ctx); err != nil {
		return fmt.Errorf("caddy: %w", err)
	}

	net := docker.ProjectNetwork(st.Tenant, st.Project)
	if err := p.docker.EnsureNetwork(ctx, net); err != nil {
		return err
	}
	if err := p.edge.ConnectProject(ctx, net); err != nil {
		return fmt.Errorf("connect caddy to project network: %w", err)
	}

	if err := p.db.EnsureProjectDBs(ctx, st.Tenant, st.Project, net, manifest.Databases); err != nil {
		return err
	}

	built, err := p.buildAll(ctx, st, manifest, workspace)
	if err != nil {
		return err
	}
	st.Services = built
	if err := p.store.WriteState(st); err != nil {
		return err
	}

	if err := p.setPhase(deployID, PhaseContainersStarted, "starting containers"); err != nil {
		return err
	}

	env, err := p.env.Resolve(st.Tenant, st.Project, manifest.Env)
	if err != nil {
		return err
	}

	newContainers, err := p.startAll(ctx, st, manifest, net, env)
	if err != nil {
		_ = p.stopContainers(ctx, newContainers)
		return err
	}
	st.NewContainers = newContainers
	if err := p.store.WriteState(st); err != nil {
		return err
	}

	if err := p.runPreDeploy(ctx, st, manifest, net, env); err != nil {
		_ = p.stopContainers(ctx, newContainers)
		return err
	}
	if err := p.setPhase(deployID, PhaseHealthcheck, "waiting for healthchecks"); err != nil {
		return err
	}
	if err := p.waitHealthy(ctx, manifest, newContainers); err != nil {
		_ = p.stopContainers(ctx, newContainers)
		return err
	}

	if err := p.setPhase(deployID, PhaseSwapped, "swapping caddy upstreams"); err != nil {
		return err
	}
	if err := p.swapCaddy(ctx, st, manifest); err != nil {
		_ = p.stopContainers(ctx, newContainers)
		return err
	}

	if err := p.setPhase(deployID, PhaseDraining, "draining old containers"); err != nil {
		return err
	}
	if err := p.drain(ctx, st.DrainContainers); err != nil {
		return err
	}

	st, _ = p.store.ReadState(deployID)
	st.Phase = PhaseSucceeded
	st.Message = "deploy succeeded"
	st.CurrentBuildID = st.BuildID
	if err := p.store.WriteState(st); err != nil {
		return err
	}
	return p.saveRuntime(st, manifest, newContainers, prev)
}

func (p *Pipeline) extractWorkspace(deployID string) error {
	ws := p.store.WorkspacePath(deployID)
	if entries, err := os.ReadDir(ws); err == nil && len(entries) > 0 {
		return nil
	}
	return archive.ExtractTarGz(p.store.TarballPath(deployID), ws)
}

func (p *Pipeline) saveRuntime(st State, m *schema.Manifest, new []ContainerRecord, prev runtime.Project) error {
	bySvc := map[string][]runtime.Container{}
	for _, c := range new {
		bySvc[c.Service] = append(bySvc[c.Service], runtime.Container{ID: c.ID, Name: c.Name})
	}
	rt := runtime.Project{
		Tenant:   st.Tenant,
		Project:  st.Project,
		BuildID:  st.BuildID,
		Services: map[string]runtime.Service{},
		Previous: copyProject(prev),
	}
	if prev.BuildID != "" {
		rt.PreviousBuildID = prev.BuildID
	}
	for _, svc := range m.Services {
		port := servicePort(svc)
		pub := svc.HTTP != nil && svc.HTTP.Public
		routeID := ""
		if pub {
			routeID = caddy.RouteID(st.Tenant, st.Project, svc.Name)
		}
		img := docker.ImageTag(st.Tenant, st.Project, svc.Name, st.BuildID)
		hosts := []string{}
		if pub {
			hosts = svc.HTTP.Domains
			if len(hosts) == 0 {
				hosts = []string{caddy.DefaultHost(st.Project, svc.Name)}
			}
		}
		rt.Services[svc.Name] = runtime.Service{
			Name:         svc.Name,
			Image:        img,
			Port:         port,
			Public:       pub,
			Hosts:        hosts,
			Containers:   bySvc[svc.Name],
			CaddyRouteID: routeID,
		}
	}
	return p.runtime.Save(rt)
}

func (p *Pipeline) buildAll(ctx context.Context, st State, m *schema.Manifest, workspace string) ([]ServiceBuild, error) {
	var built []ServiceBuild
	for _, svc := range m.Services {
		tag := docker.ImageTag(st.Tenant, st.Project, svc.Name, st.BuildID)
		p.logger.Info("docker build", "service", svc.Name, "tag", tag)
		if err := p.docker.Build(ctx, docker.BuildOpts{
			Context:    workspace,
			Dockerfile: svc.Dockerfile,
			Tag:        tag,
		}); err != nil {
			return nil, fmt.Errorf("build %s: %w", svc.Name, err)
		}
		port := svc.Port
		if port == 0 {
			port = 8080
		}
		built = append(built, ServiceBuild{Name: svc.Name, Image: tag, Port: port})
	}
	return built, nil
}

func (p *Pipeline) startAll(ctx context.Context, st State, m *schema.Manifest, network string, env []string) ([]ContainerRecord, error) {
	var records []ContainerRecord
	for _, svc := range m.Services {
		replicas := svc.Replicas
		if replicas <= 0 {
			replicas = 1
		}
		port := svc.Port
		if port == 0 {
			port = 8080
		}
		tag := docker.ImageTag(st.Tenant, st.Project, svc.Name, st.BuildID)
		cmd := splitCommand(svc.Command)
		for i := 0; i < replicas; i++ {
			name := docker.ContainerName(st.Project, svc.Name, st.BuildID, i)
			p.logger.Info("docker run", "name", name, "image", tag)
			id, err := p.docker.Run(ctx, docker.RunOpts{
				Image:   tag,
				Name:    name,
				Network: network,
				Command: cmd,
				Env:     env,
				Detach:  true,
			})
			if err != nil {
				return records, fmt.Errorf("run %s: %w", name, err)
			}
			records = append(records, ContainerRecord{ID: id, Name: name, Service: svc.Name})
		}
	}
	return records, nil
}

func (p *Pipeline) swapCaddy(ctx context.Context, st State, m *schema.Manifest) error {
	byService := map[string][]ContainerRecord{}
	for _, c := range st.NewContainers {
		byService[c.Service] = append(byService[c.Service], c)
	}
	for _, svc := range m.Services {
		if svc.HTTP == nil || !svc.HTTP.Public {
			continue
		}
		containers := byService[svc.Name]
		if len(containers) == 0 {
			return fmt.Errorf("no containers for public service %s", svc.Name)
		}
		port := svc.Port
		if port == 0 {
			port = 8080
		}
		dials := make([]string, len(containers))
		for i, c := range containers {
			dials[i] = caddy.UpstreamDial(c.Name, port)
		}
		hosts := svc.HTTP.Domains
		if len(hosts) == 0 {
			hosts = []string{caddy.DefaultHost(st.Project, svc.Name)}
		}
		routeID := caddy.RouteID(st.Tenant, st.Project, svc.Name)
		if err := p.caddy.UpsertRoute(ctx, caddy.RouteOpts{
			RouteID:   routeID,
			Hosts:     hosts,
			Upstreams: dials,
		}); err != nil {
			return fmt.Errorf("caddy route %s: %w", svc.Name, err)
		}
	}
	return nil
}

func (p *Pipeline) drain(ctx context.Context, targets []ContainerRecord) error {
	for _, c := range targets {
		p.logger.Info("drain stop", "container", c.Name)
		if err := p.docker.Stop(ctx, c.Name); err != nil {
			p.logger.Warn("stop failed", "container", c.Name, "err", err)
		}
	}
	return nil
}

func (p *Pipeline) setPhase(deployID string, phase Phase, msg string) error {
	st, err := p.store.ReadState(deployID)
	if err != nil {
		return err
	}
	st.Phase = phase
	st.Message = msg
	return p.store.WriteState(st)
}

func (p *Pipeline) fail(ctx context.Context, deployID, msg string) error {
	st, err := p.store.ReadState(deployID)
	if err != nil {
		return err
	}
	if st.Phase != PhaseSwapped && st.Phase != PhaseDraining && st.Phase != PhaseSucceeded {
		_ = p.stopContainers(ctx, st.NewContainers)
	}
	st.Phase = PhaseFailed
	st.Message = msg
	return p.store.WriteState(st)
}

func (p *Pipeline) stopContainers(ctx context.Context, containers []ContainerRecord) error {
	for _, c := range containers {
		_ = p.docker.Stop(ctx, c.Name)
		_ = p.docker.Remove(ctx, c.Name)
	}
	return nil
}

func copyProject(p runtime.Project) *runtime.Project {
	cp := p
	cp.Services = make(map[string]runtime.Service, len(p.Services))
	for k, v := range p.Services {
		containers := make([]runtime.Container, len(v.Containers))
		copy(containers, v.Containers)
		v.Containers = containers
		hosts := make([]string, len(v.Hosts))
		copy(hosts, v.Hosts)
		v.Hosts = hosts
		cp.Services[k] = v
	}
	return &cp
}

func drainTargets(prev runtime.Project) []ContainerRecord {
	var out []ContainerRecord
	for _, svc := range prev.Services {
		for _, c := range svc.Containers {
			out = append(out, ContainerRecord{
				ID:      c.ID,
				Name:    c.Name,
				Service: svc.Name,
			})
		}
	}
	return out
}

func splitCommand(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	return strings.Fields(cmd)
}
