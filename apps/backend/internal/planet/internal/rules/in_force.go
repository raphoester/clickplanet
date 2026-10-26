package rules

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type InForce struct {
	clock cptime.Clock
	set   atomic.Pointer[taken]

	mu       sync.Mutex
	watchers map[uint64]chan Set
	nextID   uint64
}

type taken struct {
	set   Set
	since time.Time
}

func NewInForce(clock cptime.Clock) *InForce {
	return &InForce{clock: clock, watchers: make(map[uint64]chan Set)}
}

func (f *InForce) Enforce(set Set) {
	f.set.Store(&taken{set: set, since: f.clock.Now()})

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

// Since is when the set in force took over.
func (f *InForce) Since() time.Time { return f.set.Load().since }

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
