package rules

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// NewInForce holds no set: until the first is enforced, every value is 0, so no click passes and no box is offered.
func NewInForce(clock cptime.Clock) *InForce {
	return &InForce{clock: clock, watchers: make(map[uint64]chan Set)}
}

type InForce struct {
	clock cptime.Clock
	set   atomic.Pointer[enforced]

	mu       sync.Mutex
	watchers map[uint64]chan Set
	nextID   uint64
}

type enforced struct {
	set   Set
	since time.Time
}

func (f *InForce) Enforce(set Set) {
	f.set.Store(&enforced{set: set, since: f.clock.Now()})

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

func (f *InForce) loaded() enforced {
	if in := f.set.Load(); in != nil {
		return *in
	}

	return enforced{}
}

func (f *InForce) Tempo() float64 {
	return f.loaded().set.tempo
}

func (f *InForce) Policy() cpratelimit.Policy {
	in := f.loaded()

	return cpratelimit.Policy{PerSecond: in.set.clicks.PerSecond, Burst: in.set.clicks.Burst, Tempo: in.set.tempo, Since: in.since}
}

func (f *InForce) Throttle() clicks.Throttle {
	return f.loaded().set.clicks
}

func (f *InForce) Toll() []clicks.TollStep {
	return f.loaded().set.toll
}

func (f *InForce) Bonuses() bonuses.Rules {
	return f.loaded().set.bonuses
}

func (f *InForce) ShieldsPerTile() int {
	return f.loaded().set.bonuses.Shield.PerTile
}

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
