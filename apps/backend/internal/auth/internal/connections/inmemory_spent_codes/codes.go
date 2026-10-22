package inmemory_spent_codes

import (
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
)

// The codes redeemed and not yet lapsed, so one is good once. In memory: a code lives a minute, so a restart forgets
// a handful, and a replay of one still needs the verifier only the app holds.
type Codes struct {
	mu    sync.Mutex
	spent map[string]time.Time
}

func New() *Codes {
	return &Codes{spent: map[string]time.Time{}}
}

func (c *Codes) Spend(id string, until, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for spent, lapses := range c.spent {
		if !now.Before(lapses) {
			delete(c.spent, spent)
		}
	}
	if _, spent := c.spent[id]; spent {
		return fmt.Errorf("%w: redeemed already", connections.ErrInvalidGrant)
	}

	c.spent[id] = until
	return nil
}
