package repo

func (r *Repo) MarkRead(id string) error     { return r.store.MarkRead(id) }
func (r *Repo) MarkSaved(id string) error    { return r.store.MarkSaved(id) }
func (r *Repo) MarkUnread(id string) error   { return r.store.MarkUnread(id) }
func (r *Repo) CountByState() map[string]int { return r.store.CountByState() }
