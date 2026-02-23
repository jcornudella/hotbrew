package repo

import "github.com/jcornudella/hotbrew/internal/store"

// Repo exposes typed access to the persistence layer.
type Repo struct {
	store *store.Store
}

func New(s *store.Store) *Repo { return &Repo{store: s} }

func (r *Repo) Store() *store.Store { return r.store }

func (r *Repo) Close() error {
	if r.store == nil {
		return nil
	}
	return r.store.Close()
}
