package repo

import "github.com/jcornudella/hotbrew/internal/store"

func (r *Repo) InsertSource(name, kind, url, icon string, settings map[string]any) (int, error) {
	return r.store.InsertSource(name, kind, url, icon, settings)
}

func (r *Repo) GetOrCreateSource(name, kind, url, icon string) (int, error) {
	return r.store.GetOrCreateSource(name, kind, url, icon)
}

func (r *Repo) ListSources() ([]store.SourceRecord, error) {
	return r.store.ListSources()
}

func (r *Repo) UpdateLastSync(id int) error { return r.store.UpdateLastSync(id) }
