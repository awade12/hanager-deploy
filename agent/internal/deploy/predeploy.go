package deploy

import (
	"context"
	"fmt"

	"github.com/awade12/hanager-deploy/agent/internal/docker"
	"github.com/awade12/hanager-deploy/pkg/schema"
)

func (p *Pipeline) runPreDeploy(ctx context.Context, st State, m *schema.Manifest, network string, env []string) error {
	if m.Deploy.PreDeploy == "" {
		return nil
	}
	svcName := m.Deploy.PreDeployService
	if svcName == "" {
		return fmt.Errorf("pre_deploy_service required when pre_deploy is set")
	}
	var svc *schema.Service
	for i := range m.Services {
		if m.Services[i].Name == svcName {
			svc = &m.Services[i]
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("pre_deploy_service %q not found", svcName)
	}
	image := docker.ImageTag(st.Tenant, st.Project, svc.Name, st.BuildID)
	cmd := splitCommand(m.Deploy.PreDeploy)
	p.logger.Info("pre_deploy", "service", svcName, "cmd", m.Deploy.PreDeploy)
	if err := p.docker.RunOnce(ctx, docker.RunOnceOpts{
		Image:   image,
		Network: network,
		Command: cmd,
		Env:     env,
	}); err != nil {
		return fmt.Errorf("pre_deploy: %w", err)
	}
	return nil
}
