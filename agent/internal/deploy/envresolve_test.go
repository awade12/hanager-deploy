package deploy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awade12/hanager-deploy/agent/internal/database"
	"github.com/awade12/hanager-deploy/agent/internal/deploy"
	"github.com/awade12/hanager-deploy/agent/internal/secret"
)

func TestEnvResolver(t *testing.T) {
	dir := t.TempDir()
	sec, err := secret.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := sec.Set("default", "api_key", "secret-value"); err != nil {
		t.Fatal(err)
	}
	dbSvc := database.NewService(dir, nil)
	env := map[string]string{
		"PLAIN": "ok",
		"KEY":   "$secret:api_key",
	}
	r := deploy.NewEnvResolver(sec, dbSvc)
	out, err := r.Resolve("default", "proj", env)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}

func TestSecretStoreEncryptRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s, err := secret.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("t", "k", "v"); err != nil {
		t.Fatal(err)
	}
	s2, err := secret.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	v, ok, err := s2.Get("t", "k")
	if err != nil || !ok || v != "v" {
		t.Fatalf("got %q ok=%v err=%v", v, ok, err)
	}
	enc := filepath.Join(dir, "secrets", "t.enc")
	if st, err := os.Stat(enc); err != nil || st.Size() == 0 {
		t.Fatalf("encrypted file: %v", err)
	}
}
