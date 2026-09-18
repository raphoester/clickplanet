package bonuses

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Holder names who holds a charge: the account when the caller has one, else its scope. A charge is the
// account's so it follows the player across tabs, addresses and devices, and waits for them to come back.
type Holder string

// HolderOf reads the holder off the payer the throttle already derives, so the claim, the click and the
// drop cannot disagree on whose charge it is.
func HolderOf(payer clicks.Payer) Holder {
	if payer.Account == "" {
		return Holder("scope:" + payer.Scope)
	}

	return Holder("account:" + payer.Account)
}

// Held is what one holder has in hand: at most one charge of each kind.
type Held struct {
	Bomb    bool
	Enclose bool

	// The clicks left on a spread charge. Zero is no spread charge.
	SpreadClicks int
}

// Kinds is every kind held, which the schedule does not offer again until it is spent.
func (h Held) Kinds() []Kind {
	var kinds []Kind
	if h.Bomb {
		kinds = append(kinds, KindBomb)
	}
	if h.Enclose {
		kinds = append(kinds, KindEncloseClicks)
	}
	if h.SpreadClicks > 0 {
		kinds = append(kinds, KindSpreadClicks)
	}

	return kinds
}

// Charges holds the use-once bonuses: a bomb, an enclose shape, and a spread's clicks. None of them runs
// on a clock; each is kept until it is spent or until ChargeTTL has passed since it was granted.
//
// They live in memory, like every other piece of bonus state: a restart loses every charge held.
//
// Nothing is pushed when they change. The client reads them once (GetCharges) and follows its own calls:
// what a claim grants, a drop spends, an accepted click spends of a spread, and its own shape spends of an
// enclose. So a click never answers with them, which would tell a shadow-banned caller its clicks spend nothing.
type Charges struct {
	config ChargesConfig
	clock  cptime.Clock

	mu   sync.Mutex
	held map[Holder]*hand
}

// ChargesConfig is the part of Config the charges read.
type ChargesConfig struct {
	TTL               time.Duration
	SpreadClicks      int
	EnclosureMaxTiles int
}

// hand is one holder's charges, each with the moment it lapses. A zero time is no charge.
type hand struct {
	bomb    time.Time
	enclose time.Time
	spread  time.Time

	spreadClicks int
}

func NewCharges(config ChargesConfig, clock cptime.Clock) *Charges {
	return &Charges{config: config, clock: clock, held: make(map[Holder]*hand)}
}

// EnclosureMaxTiles is the most tiles one enclosed shape may hold.
func (c *Charges) EnclosureMaxTiles() int {
	return c.config.EnclosureMaxTiles
}

// SpreadClicks is how many clicks one spread charge spreads.
func (c *Charges) SpreadClicks() int {
	return c.config.SpreadClicks
}

// Grant hands holder one charge of kind. A second of a kind already held replaces it rather than adding
// to it: nobody holds two, which is what stops a stockpile being dropped all at once. A timed kind is not
// a charge and grants nothing here. It also forgets every hand that has nothing left, so it needs no sweep.
func (c *Charges) Grant(holder Holder, kind Kind) {
	now := c.clock.Now()
	lapses := now.Add(c.config.TTL)

	c.mu.Lock()
	defer c.mu.Unlock()

	for other, h := range c.held {
		if h.empty(now) {
			delete(c.held, other)
		}
	}

	h, ok := c.held[holder]
	if !ok {
		h = &hand{}
		c.held[holder] = h
	}

	switch kind {
	case KindBomb:
		h.bomb = lapses
	case KindEncloseClicks:
		h.enclose = lapses
	case KindSpreadClicks:
		h.spread = lapses
		h.spreadClicks = c.config.SpreadClicks
	case KindTripleClicks:
	}
}

// Held is what holder has in hand right now.
func (c *Charges) Held(holder Holder) Held {
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	h, ok := c.held[holder]
	if !ok {
		return Held{}
	}

	return h.held(now)
}

// SpendBomb takes holder's bomb, and reports whether there was one: two drops racing for it get one bomb.
func (c *Charges) SpendBomb(holder Holder) bool {
	return c.spend(holder, func(h *hand, now time.Time) bool {
		if !now.Before(h.bomb) {
			return false
		}
		h.bomb = time.Time{}

		return true
	})
}

// SpendEnclose takes holder's enclose charge, one shape, and reports whether there was one.
func (c *Charges) SpendEnclose(holder Holder) bool {
	return c.spend(holder, func(h *hand, now time.Time) bool {
		if !now.Before(h.enclose) {
			return false
		}
		h.enclose = time.Time{}

		return true
	})
}

// SpendSpreadClick takes one click off holder's spread charge, and reports whether there was one.
func (c *Charges) SpendSpreadClick(holder Holder) bool {
	return c.spend(holder, func(h *hand, now time.Time) bool {
		if h.spreadClicks <= 0 || !now.Before(h.spread) {
			return false
		}
		h.spreadClicks--
		if h.spreadClicks == 0 {
			h.spread = time.Time{}
		}

		return true
	})
}

// spend runs take under the lock.
func (c *Charges) spend(holder Holder, take func(h *hand, now time.Time) bool) bool {
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	h, ok := c.held[holder]

	return ok && take(h, now)
}

func (h *hand) held(now time.Time) Held {
	held := Held{Bomb: now.Before(h.bomb), Enclose: now.Before(h.enclose)}
	if now.Before(h.spread) {
		held.SpreadClicks = h.spreadClicks
	}

	return held
}

func (h *hand) empty(now time.Time) bool {
	return h.held(now) == (Held{})
}
