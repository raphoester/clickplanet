package bonus

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Enclosures holds the running enclose bonuses, by scope.
type Enclosures struct {
	clock cptime.Clock

	mu      sync.Mutex
	running map[string]*Enclosure
}

func NewEnclosures(clock cptime.Clock) *Enclosures {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Enclosures{clock: clock, running: make(map[string]*Enclosure)}
}

// Grant also forgets every bonus that is over, so the map needs no sweep.
func (e *Enclosures) Grant(scope string, until time.Time, shapes int, maxTiles int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for other, enclosure := range e.running {
		if !enclosure.Running() {
			delete(e.running, other)
		}
	}

	e.running[scope] = &Enclosure{clock: e.clock, until: until, left: shapes, maxTiles: maxTiles}
}

func (e *Enclosures) Running(scope string) (*Enclosure, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	enclosure, found := e.running[scope]
	if !found || !enclosure.Running() {
		return nil, false
	}

	return enclosure, true
}

// Enclosure ends with its time or its last shape, whichever comes first.
type Enclosure struct {
	clock    cptime.Clock
	until    time.Time
	maxTiles int

	mu   sync.Mutex
	left int
}

func (e *Enclosure) MaxTiles() int {
	return e.maxTiles
}

func (e *Enclosure) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.runningLocked()
}

// Spend settles two clicks racing for the last shape: only one gets it.
func (e *Enclosure) Spend() (left int, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.runningLocked() {
		return 0, false
	}

	e.left--

	return e.left, true
}

func (e *Enclosure) runningLocked() bool {
	return e.left > 0 && e.clock.Now().Before(e.until)
}
