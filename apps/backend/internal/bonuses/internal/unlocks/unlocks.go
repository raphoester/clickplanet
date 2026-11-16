package unlocks

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

var ErrNoTime = errors.New("the decision says no time")

// Unlocks is what progression last decided an account may be drawn beyond the kinds open to everyone. An account it
// never told of has none, as a guest.
type Unlocks struct {
	kinds     bonuses.Unlocked
	decidedAt time.Time
}

// Decision is every kind progression unlocked for an account at a time: the whole list, never one change.
type Decision struct {
	kinds bonuses.Unlocked
	at    time.Time
}

func DecisionOf(kinds bonuses.Unlocked, at time.Time) Decision {
	return Decision{kinds: kinds, at: at}
}

func (d Decision) TimeError() error {
	if d.at.IsZero() {
		return ErrNoTime
	}
	return nil
}

// Only a later decision changes the unlocks: a raise and a reconciliation tell them, late, twice or out of order.
func (u Unlocks) Decided(decision Decision) Unlocks {
	if !decision.at.After(u.decidedAt) {
		return u
	}
	kinds := slices.Clone(decision.kinds)
	slices.Sort(kinds)
	return Unlocks{kinds: slices.Compact(kinds), decidedAt: decision.at}
}

func (u Unlocks) TakeSnapshot() Snapshot {
	return Snapshot{Kinds: slices.Clone(u.kinds), DecidedAt: u.decidedAt}
}

type Snapshot struct {
	Kinds     bonuses.Unlocked
	DecidedAt time.Time
}

func (s Snapshot) Restore() Unlocks {
	return Unlocks{kinds: slices.Clone(s.Kinds), decidedAt: s.DecidedAt}
}

type Store interface {
	Change(ctx context.Context, account bonuses.Account, change func(Unlocks) Unlocks) error
	// Unlocked is the kinds kept for an account: none for one never told of.
	Unlocked(ctx context.Context, account bonuses.Account) (bonuses.Unlocked, error)
	Forget(ctx context.Context, account bonuses.Account) error
}
