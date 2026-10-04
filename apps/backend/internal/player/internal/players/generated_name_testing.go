//go:build testing

package players

import "sync"

type RepeatedNames struct {
	mu    sync.Mutex
	names []Name
	drawn int
}

func NewRepeatedNames(names ...Name) *RepeatedNames {
	return &RepeatedNames{names: names}
}

func (r *RepeatedNames) NewName() (Name, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.drawn++
	name := r.names[0]
	if len(r.names) > 1 {
		r.names = r.names[1:]
	}
	return name, nil
}

func (r *RepeatedNames) Drawn() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.drawn
}
