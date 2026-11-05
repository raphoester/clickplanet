package inmemory_spent_codes

import (
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
)

// In memory is enough: a code lives a minute, and a replay still needs the verifier.
type Codes struct {
	mu    sync.Mutex
	spent map[connections.CodeID]time.Time
}

func New() *Codes {
	return &Codes{spent: map[connections.CodeID]time.Time{}}
}

func (c *Codes) Spend(code *connections.Code, now time.Time) error {
	redeemed := code.TakeSnapshot()
	c.mu.Lock()
	defer c.mu.Unlock()

	for spent, lapses := range c.spent {
		if !now.Before(lapses) {
			delete(c.spent, spent)
		}
	}
	if _, spent := c.spent[redeemed.ID]; spent {
		return fmt.Errorf("%w: redeemed already", connections.ErrInvalidGrant)
	}

	c.spent[redeemed.ID] = redeemed.ExpiresAt
	return nil
}
