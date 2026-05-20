package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hangar-sh/hangar/pkg/fsutil"
)

type Store struct {
	root string
}

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Dir(id string) string {
	return filepath.Join(s.root, id)
}

func (s *Store) StatePath(id string) string {
	return filepath.Join(s.Dir(id), "state.json")
}

func (s *Store) TarballPath(id string) string {
	return filepath.Join(s.Dir(id), "source.tar.gz")
}

func (s *Store) WorkspacePath(id string) string {
	return filepath.Join(s.Dir(id), "workspace")
}

func (s *Store) TOMLPath(id string) string {
	return filepath.Join(s.Dir(id), "hangar.toml")
}

func (s *Store) Ensure(id string) error {
	return os.MkdirAll(s.Dir(id), 0o755)
}

func (s *Store) WriteState(st State) error {
	if err := s.Ensure(st.ID); err != nil {
		return err
	}
	st.UpdatedAt = time.Now().UTC()
	return fsutil.WriteJSONAtomic(s.StatePath(st.ID), st)
}

func (s *Store) ReadState(id string) (State, error) {
	var st State
	err := fsutil.ReadJSON(s.StatePath(id), &st)
	return st, err
}

func (s *Store) ListInFlight() ([]State, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []State
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := s.ReadState(e.Name())
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read state %s: %w", e.Name(), err)
		}
		if st.IsInFlight() || st.Phase == PhasePending {
			out = append(out, st)
		}
	}
	return out, nil
}

func (s *Store) ListAll() ([]State, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []State
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := s.ReadState(e.Name())
		if err != nil {
			continue
		}
		out = append(out, st)
	}
	return out, nil
}
