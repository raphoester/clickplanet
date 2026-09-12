package bonus

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Bombs remembers who holds an undropped bomb and until when, the way Spreads does for spreads.
type Bombs struct {
	clock cptime.Clock

	mu    sync.Mutex
	until map[string]time.Time
}

func NewBombs(clock cptime.Clock) *Bombs {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Bombs{clock: clock, until: make(map[string]time.Time)}
}

// Grant hands scope a bomb, replacing any it held, and forgets the lapsed ones on the way.
func (b *Bombs) Grant(scope string, until time.Time) {
	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	for other, end := range b.until {
		if !now.Before(end) {
			delete(b.until, other)
		}
	}

	b.until[scope] = until
}

// Take spends scope's bomb, and reports whether there was one to spend.
func (b *Bombs) Take(scope string) bool {
	now := b.clock.Now()

	b.mu.Lock()
	defer b.mu.Unlock()

	end, ok := b.until[scope]
	delete(b.until, scope)

	return ok && now.Before(end)
}
