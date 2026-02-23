package repo

import (
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/pkg/trss"
)

func (r *Repo) InsertItem(item trss.Item, sourceID int) error {
	return r.store.InsertItem(item, sourceID)
}

func (r *Repo) ListItems(filter store.ItemFilter) ([]trss.Item, error) {
	return r.store.ListItems(filter)
}

func (r *Repo) GetItem(idPrefix string) (*trss.Item, error) {
	return r.store.GetItem(idPrefix)
}

func (r *Repo) UpdateItemScore(id string, score float64) error {
	return r.store.UpdateScore(id, score)
}

func (r *Repo) ItemCount() int {
	return r.store.ItemCount()
}
