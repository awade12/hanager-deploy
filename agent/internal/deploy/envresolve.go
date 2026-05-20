package deploy

import (
	"fmt"
	"strings"

	"github.com/hangar-sh/hangar/agent/internal/database"
	"github.com/hangar-sh/hangar/agent/internal/secret"
	"github.com/hangar-sh/hangar/pkg/schema"
)

type EnvResolver struct {
	secrets *secret.Store
	db      *database.Service
}

func NewEnvResolver(secrets *secret.Store, db *database.Service) *EnvResolver {
	return &EnvResolver{secrets: secrets, db: db}
}

func (r *EnvResolver) Resolve(tenant, project string, env map[string]string) ([]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		resolved, err := r.resolveValue(tenant, project, v)
		if err != nil {
			return nil, fmt.Errorf("env.%s: %w", k, err)
		}
		out = append(out, k+"="+resolved)
	}
	return out, nil
}

func (r *EnvResolver) resolveValue(tenant, project, value string) (string, error) {
	if !strings.HasPrefix(value, "$") {
		return value, nil
	}
	ref, ok := schema.ParseEnvValue(value)
	if !ok {
		return "", fmt.Errorf("invalid ref %q", value)
	}
	switch ref.Kind {
	case schema.RefSecret:
		return r.secrets.Resolve(tenant, ref.SecretName)
	case schema.RefDB:
		if ref.TenantDB {
			return r.db.ResolveURL(tenant, project, ref.DBName, true)
		}
		return r.db.ResolveURL(tenant, project, ref.DBName, false)
	default:
		return "", fmt.Errorf("unknown ref %q", value)
	}
}
