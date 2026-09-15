//go:build testing

package cptime

import (
	"sync"
	"time"
)

// FixedClock is the Clock tests hand to production code: it stands still until
// the test moves it, which is what makes a window, a cooldown or an expiry
// assertable instead of flaky. Twelve test packages need one, so it lives
// behind the tag rather than being redeclared in each of them.
//
// It locks on every read because some of the code under test reads the time
// from goroutines of its own — the storage sweep and the rate limiter both do.
type FixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFixedClock(now time.Time) *FixedClock {
	return &FixedClock{now: now}
}

func (c *FixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d. A negative d moves it backwards, which
// is how a test says the wall clock jumped the wrong way.
func (c *FixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var _ Clock = (*FixedClock)(nil)
