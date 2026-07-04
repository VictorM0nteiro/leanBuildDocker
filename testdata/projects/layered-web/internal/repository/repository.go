package repository

// Repo is the deepest layer — plain in-memory storage. Its only purpose in
// the fixture is to prove the analyzer reaches code nested several packages
// below the entry point.
type Repo struct {
	items map[string]string
}

func New() *Repo {
	return &Repo{items: map[string]string{}}
}

func (r *Repo) Set(k, v string) { r.items[k] = v }

func (r *Repo) Get(k string) string { return r.items[k] }
