package rules

import (
	"sync"
	"sync/atomic"
)

// InForce is the set the offers play, which every reader reads at each use, and the streams that hear each set taken.
type InForce struct {
	set atomic.Pointer[taken]

	mu       sync.Mutex
	watchers map[uint64]chan Set
	nextID   uint64
}

type taken struct {
	set Set
}

func NewInForce() *InForce {
	return &InForce{watchers: make(map[uint64]chan Set)}
}

func (f *InForce) Enforce(set Set) {
	f.set.Store(&taken{set: set})

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, watcher := range f.watchers {
		select {
		case <-watcher:
		default:
		}
		watcher <- set
	}
}

func (f *InForce) Name() SetName { return f.set.Load().set.name }

// Watch hands over each set enforced from now on; one not taken yet is replaced by the next.
func (f *InForce) Watch() (<-chan Set, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	id := f.nextID
	watcher := make(chan Set, 1)
	f.watchers[id] = watcher

	return watcher, func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		delete(f.watchers, id)
	}
}
