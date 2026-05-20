package schema_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/awade12/hanager-deploy/pkg/schema"
)

func TestParseValidFixture(t *testing.T) {
	path := filepath.Join("testdata", "valid.toml")
	m, err := schema.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Project.Name != "chronalife" {
		t.Fatalf("project name = %q", m.Project.Name)
	}
	if len(m.Services) != 2 {
		t.Fatalf("services = %d", len(m.Services))
	}
}

func TestSchemaMustBeOne(t *testing.T) {
	_, err := schema.Parse([]byte(`schema = 2\n[project]\nname = "x"\n[[service]]\nname = "a"\ndockerfile = "D"\nport = 1`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDuplicateServiceName(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "api"
dockerfile = "./D"
port = 3000
[[service]]
name = "api"
dockerfile = "./D"
port = 3001
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestPortConflict(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "a"
dockerfile = "./D"
port = 3000
[[service]]
name = "b"
dockerfile = "./D"
port = 3000
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "port 3000") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestUnknownDBRef(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "api"
dockerfile = "./D"
port = 3000
[env]
DATABASE_URL = "$db:missing"
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestInvalidDollarRef(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "api"
dockerfile = "./D"
port = 3000
[env]
BAD = "$db:"
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInlineSecretRejected(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "api"
dockerfile = "./D"
port = 3000
[env]
API_SECRET = "super-secret-value"
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "$secret:") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestPreDeployUnknownService(t *testing.T) {
	toml := `
schema = 1
[project]
name = "p"
[[service]]
name = "api"
dockerfile = "./D"
port = 3000
[deploy]
pre_deploy = "migrate"
pre_deploy_service = "worker"
`
	_, err := schema.Parse([]byte(toml))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEnvRef(t *testing.T) {
	ref, ok := schema.ParseEnvValue("$db:tenant/devpg")
	if !ok || !ref.TenantDB || ref.DBName != "devpg" {
		t.Fatalf("got %+v ok=%v", ref, ok)
	}
	ref, ok = schema.ParseEnvValue("$secret:sentry_dsn")
	if !ok || ref.SecretName != "sentry_dsn" {
		t.Fatalf("got %+v ok=%v", ref, ok)
	}
}
