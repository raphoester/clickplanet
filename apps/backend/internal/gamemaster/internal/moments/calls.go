package moments

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Calls is what a process knows of the calls it made in order: each answered for good, and when one that failed is made
// again. A call not answered waits for its turn, and holds back every call after it.
type Calls[K comparable] struct {
	answered *cpcolls.Set[K]
	again    map[K]time.Time
	wait     time.Duration
}

// NewCalls makes a failed call again after wait.
func NewCalls[K comparable](wait time.Duration) *Calls[K] {
	return &Calls[K]{answered: cpcolls.NewSet[K](), again: map[K]time.Time{}, wait: wait}
}

// Answered is true for a call answered for good in this process: kept, refused, or open.
func (c *Calls[K]) Answered(key K) bool {
	return c.answered.Contains(key)
}

// Waits is true while a call that failed waits for its next try.
func (c *Calls[K]) Waits(key K, now time.Time) bool {
	return now.Before(c.again[key])
}

func (c *Calls[K]) Answer(key K) {
	c.answered.Add(key)
	delete(c.again, key)
}

func (c *Calls[K]) Fail(key K, now time.Time) {
	c.again[key] = now.Add(c.wait)
}
