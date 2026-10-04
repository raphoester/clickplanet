//go:build testing

package players

import "sync"

type RepeatedNames struct {
	mu    sync.Mutex
	names []Name
}

func NewRepeatedNames(names ...Name) *RepeatedNames {
	return &RepeatedNames{names: names}
}

func (r *RepeatedNames) NewName() (Name, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := r.names[0]
	if len(r.names) > 1 {
		r.names = r.names[1:]
	}
	return name, nil
}
