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
	SetFrozen(frozen bool)
}

type Guard interface {
	SetCrowd(members int)
}

// NewInForce holds set without telling tempo, schedule or guard: they start from the same config.
func NewInForce(set Set, tempo Tempo, schedule Schedule, guard Guard) *InForce {
	return &InForce{set: set, tempo: tempo, schedule: schedule, guard: guard, watchers: make(map[uint64]chan Set)}
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
	f.tempo.SetTempo(set.Tempo)
	f.schedule.SetPace(set.Boxes)
	f.schedule.SetFrozen(set.Frozen)
	f.guard.SetCrowd(set.AntiBot.Crowd)

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

	return f.set.Tempo
}

// Watch hands over each set enforced from now on; one not taken yet is replaced by the next.
func (f *InForce) Frozen() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.set.Frozen
}

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
