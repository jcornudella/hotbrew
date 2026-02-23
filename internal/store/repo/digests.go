package repo

import "github.com/jcornudella/hotbrew/pkg/trss"

func (r *Repo) SaveDigest(d *trss.Digest) error {
	return r.store.SaveDigest(d)
}

func (r *Repo) LatestDigest() (*trss.Digest, error) {
	return r.store.GetLatestDigest()
}
