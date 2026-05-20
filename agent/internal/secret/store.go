package secret

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

)

type Store struct {
	root string
	key  []byte
	mu   sync.RWMutex
}

func Open(dataDir string) (*Store, error) {
	keyPath := filepath.Join(dataDir, "secrets.key")
	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		return nil, err
	}
	return &Store{
		root: filepath.Join(dataDir, "secrets"),
		key:  key,
	}, nil
}

func (s *Store) path(tenant string) string {
	return filepath.Join(s.root, sanitize(tenant)+".enc")
}

func (s *Store) load(tenant string) (map[string]string, error) {
	path := s.path(tenant)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return decrypt(s.key, data)
}

func (s *Store) save(tenant string, secrets map[string]string) error {
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	blob, err := encrypt(s.key, secrets)
	if err != nil {
		return err
	}
	tmp := s.path(tenant) + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(tenant))
}

func (s *Store) Set(tenant, name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.load(tenant)
	if err != nil {
		return err
	}
	m[name] = value
	return s.save(tenant, m)
}

func (s *Store) Get(tenant, name string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, err := s.load(tenant)
	if err != nil {
		return "", false, err
	}
	v, ok := m[name]
	return v, ok, nil
}

func (s *Store) List(tenant string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, err := s.load(tenant)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out, nil
}

func (s *Store) Delete(tenant, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.load(tenant)
	if err != nil {
		return err
	}
	delete(m, name)
	return s.save(tenant, m)
}

func (s *Store) Resolve(tenant, name string) (string, error) {
	v, ok, err := s.Get(tenant, name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("secret %q not found", name)
	}
	return v, nil
}

func sanitize(t string) string {
	out := make([]byte, 0, len(t))
	for i := 0; i < len(t); i++ {
		c := t[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return string(out)
}
