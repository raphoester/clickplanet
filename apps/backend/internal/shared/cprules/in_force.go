package cprules

import (
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// InForce is the set a module plays, which every reader reads at each use, and the streams that hear each set taken.
type InForce[M proto.Message] struct {
	clock cptime.Clock
	set   atomic.Pointer[taken[M]]

	mu       sync.Mutex
	watchers map[uint64]chan Set[M]
	nextID   uint64
}

type taken[M proto.Message] struct {
	set   Set[M]
	since time.Time
}

func NewInForce[M proto.Message](clock cptime.Clock) *InForce[M] {
	return &InForce[M]{clock: clock, watchers: make(map[uint64]chan Set[M])}
}

func (f *InForce[M]) Enforce(set Set[M]) {
	f.set.Store(&taken[M]{set: set, since: f.clock.Now()})

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

func (f *InForce[M]) Name() SetName { return f.set.Load().set.name }

// Since is when the set in force took over.
func (f *InForce[M]) Since() time.Time { return f.set.Load().since }

// Watch hands over each set enforced from now on; one not taken yet is replaced by the next.
func (f *InForce[M]) Watch() (<-chan Set[M], func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	id := f.nextID
	watcher := make(chan Set[M], 1)
	f.watchers[id] = watcher

	return watcher, func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		delete(f.watchers, id)
	}
}
