package rules

import (
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type Tempo interface {
	SetTempo(tempo float64)
}

type Schedule interface {
	SetPace(pace bonuses.Pace)
}

type Guard interface {
	SetCrowd(members int)
}

// NewInForce holds no set: until the first is enforced, the limiter is at its own tempo, no box is offered and no
// crowd is banned.
func NewInForce(tempo Tempo, schedule Schedule, guard Guard) *InForce {
	return &InForce{tempo: tempo, schedule: schedule, guard: guard, watchers: make(map[uint64]chan Set)}
}

type InForce struct {
	tempo    Tempo
	schedule Schedule
	guard    Guard

	mu       sync.Mutex
	set      Set
	watchers map[uint64]chan Set
	nextID   uint64
}

func (f *InForce) Enforce(set Set) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.set = set
	f.tempo.SetTempo(set.tempo)
	f.schedule.SetPace(set.boxes)
	f.guard.SetCrowd(set.crowd)

	for _, watcher := range f.watchers {
		select {
		case <-watcher:
		default:
		}
		watcher <- set
	}
}

func (f *InForce) Tempo() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.set.tempo
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
