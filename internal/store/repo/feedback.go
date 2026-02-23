package repo

func (r *Repo) InsertFeedback(rating int, note string) error {
	return r.store.InsertFeedback(rating, note)
}
