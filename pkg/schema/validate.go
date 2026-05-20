package schema

import (
	"fmt"
	"slices"
	"strings"
)

var (
	allowedEngines   = []string{"postgres", "redis"}
	allowedStrategies = []string{"rolling", "bluegreen", "recreate", ""}
)

func Validate(m *Manifest) error {
	var errs ValidationErrors

	if m.Schema != 1 {
		errs = append(errs, ValidationError{Path: "schema", Message: "must be 1"})
	}
	if strings.TrimSpace(m.Project.Name) == "" {
		errs = append(errs, ValidationError{Path: "project.name", Message: "required"})
	}
	if len(m.Services) == 0 {
		errs = append(errs, ValidationError{Path: "service", Message: "at least one service required"})
	}

	serviceNames := make(map[string]struct{}, len(m.Services))
	ports := make(map[int]string)

	for i, svc := range m.Services {
		prefix := fmt.Sprintf("service[%d]", i)
		name := strings.TrimSpace(svc.Name)
		if name == "" {
			errs = append(errs, ValidationError{Path: prefix + ".name", Message: "required"})
			continue
		}
		if _, dup := serviceNames[name]; dup {
			errs = append(errs, ValidationError{Path: prefix + ".name", Message: fmt.Sprintf("duplicate service %q", name)})
		}
		serviceNames[name] = struct{}{}

		if strings.TrimSpace(svc.Dockerfile) == "" {
			errs = append(errs, ValidationError{Path: prefix + ".dockerfile", Message: "required"})
		}
		if svc.Port < 0 || svc.Port > 65535 {
			errs = append(errs, ValidationError{Path: prefix + ".port", Message: "must be 0-65535"})
		}
		if svc.Port > 0 {
			if other, taken := ports[svc.Port]; taken {
				errs = append(errs, ValidationError{
					Path:    prefix + ".port",
					Message: fmt.Sprintf("port %d already used by service %q", svc.Port, other),
				})
			} else {
				ports[svc.Port] = name
			}
		}
		if svc.Replicas < 0 {
			errs = append(errs, ValidationError{Path: prefix + ".replicas", Message: "must be >= 0"})
		}
		if svc.Replicas == 0 && svc.Name != "" {
			// zero means unset in toml; default to 1 at runtime later
		}
	}

	dbNames := make(map[string]struct{}, len(m.Databases))
	tenantDB := make(map[string]struct{})
	for i, db := range m.Databases {
		prefix := fmt.Sprintf("database[%d]", i)
		name := strings.TrimSpace(db.Name)
		if name == "" {
			errs = append(errs, ValidationError{Path: prefix + ".name", Message: "required"})
			continue
		}
		if _, dup := dbNames[name]; dup {
			errs = append(errs, ValidationError{Path: prefix + ".name", Message: fmt.Sprintf("duplicate database %q", name)})
		}
		dbNames[name] = struct{}{}
		if db.Engine != "" && !slices.Contains(allowedEngines, db.Engine) {
			errs = append(errs, ValidationError{
				Path:    prefix + ".engine",
				Message: fmt.Sprintf("must be one of %v", allowedEngines),
			})
		}
	}

	strategy := m.Deploy.Strategy
	if strategy == "" {
		strategy = "rolling"
	}
	if !slices.Contains(allowedStrategies, m.Deploy.Strategy) && m.Deploy.Strategy != "" {
		errs = append(errs, ValidationError{
			Path:    "deploy.strategy",
			Message: fmt.Sprintf("must be one of rolling, bluegreen, recreate"),
		})
	}
	if m.Deploy.PreDeploy != "" {
		svc := strings.TrimSpace(m.Deploy.PreDeployService)
		if svc == "" {
			errs = append(errs, ValidationError{
				Path:    "deploy.pre_deploy_service",
				Message: "required when pre_deploy is set",
			})
		} else if _, ok := serviceNames[svc]; !ok {
			errs = append(errs, ValidationError{
				Path:    "deploy.pre_deploy_service",
				Message: fmt.Sprintf("unknown service %q", svc),
			})
		}
	}

	for key, value := range m.Env {
		path := "env." + key
		if IsDollarRef(value) {
			ref, ok := ParseEnvValue(value)
			if !ok {
				errs = append(errs, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("invalid ref %q", value),
				})
				continue
			}
			switch ref.Kind {
			case RefDB:
				if ref.TenantDB {
					tenantDB[ref.DBName] = struct{}{}
				} else if _, ok := dbNames[ref.DBName]; !ok {
					errs = append(errs, ValidationError{
						Path:    path,
						Message: fmt.Sprintf("$db:%s not defined in [[database]]", ref.DBName),
					})
				}
			case RefSecret:
				if strings.TrimSpace(ref.SecretName) == "" {
					errs = append(errs, ValidationError{Path: path, Message: "empty secret name"})
				}
			}
			continue
		}
		if looksLikeInlineSecret(key, value) {
			errs = append(errs, ValidationError{
				Path:    path,
				Message: "use $secret:name instead of inline secret values",
			})
		}
	}

	_ = tenantDB

	return errs.OrNil()
}

func looksLikeInlineSecret(key, value string) bool {
	upper := strings.ToUpper(key)
	if !strings.Contains(upper, "SECRET") &&
		!strings.Contains(upper, "TOKEN") &&
		!strings.Contains(upper, "PASSWORD") &&
		!strings.HasSuffix(upper, "_KEY") {
		return false
	}
	if IsDollarRef(value) {
		return false
	}
	return strings.TrimSpace(value) != ""
}
