package bonus

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Enclosures remembers who holds a running enclose bonus, and how many shapes it
// may still close. The claim starts one, and the click chain spends it.
//
// Unlike a spread it can end before its time: a bonus that has closed all its
// shapes is over, whatever the clock says.
type Enclosures struct {
	clock cptime.Clock

	mu      sync.Mutex
	running map[string]*enclosure
}

type enclosure struct {
	until    time.Time
	left     int
	maxTiles int
}

func NewEnclosures(clock cptime.Clock) *Enclosures {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Enclosures{clock: clock, running: make(map[string]*enclosure)}
}

// Grant starts an enclose bonus for scope: until the given time, at most
// `shapes` shapes, each holding at most maxTiles tiles.
//
// It also forgets every bonus that has run out, the way Spreads.Grant does.
func (e *Enclosures) Grant(scope string, until time.Time, shapes int, maxTiles int) {
	now := e.clock.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	for other, running := range e.running {
		if !running.live(now) {
			delete(e.running, other)
		}
	}

	e.running[scope] = &enclosure{until: until, left: shapes, maxTiles: maxTiles}
}

// Enclosing reports whether scope's clicks may close a shape right now, and the
// most tiles that shape may hold.
func (e *Enclosures) Enclosing(scope string) (maxTiles int, ok bool) {
	now := e.clock.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	running, found := e.running[scope]
	if !found || !running.live(now) {
		return 0, false
	}

	return running.maxTiles, true
}

// Spend uses one shape. It fails when the bonus has run out or has no shape
// left, and otherwise says how many are left after this one.
//
// Finding a shape and spending it are two steps, so two clicks at once can both
// find one while a single shape is left. Spend is what settles it: only one of
// them gets it.
func (e *Enclosures) Spend(scope string) (left int, ok bool) {
	now := e.clock.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	running, found := e.running[scope]
	if !found || !running.live(now) {
		return 0, false
	}

	running.left--

	return running.left, true
}

func (e *enclosure) live(now time.Time) bool {
	return e.left > 0 && now.Before(e.until)
}
