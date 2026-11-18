// Package pastnames keeps every username a player chose to leave for another: the ledger its public card reads its
// last names from (GetPlayer), so the others can tell who a player was. Kept since 2026-11-17, until the account is
// forgotten, and written after the save, from player.v1.NameLeft. A name an operator replaced is not one: that is the
// renames' history, the operator's alone, and a name taken off for its words must not come back on the card.
package pastnames

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	ErrNoName = errors.New("a past name with no name")
	ErrNoTime = errors.New("a past name with no time")
)

type PastName struct {
	snapshot Snapshot
}

type Snapshot struct {
	Account players.AccountID
	Name    players.Name
	LeftAt  time.Time
}

// The name the profile held before its player chose another, false when it left none: a first name replaces nothing,
// and the same name in another case is the same name.
func PastNameOf(before players.Profile, after players.Profile, at time.Time) (PastName, bool) {
	old, renamed := before.TakeSnapshot(), after.TakeSnapshot()
	if old.Name == "" || old.Name.Folded() == renamed.Name.Folded() {
		return PastName{}, false
	}
	return PastName{snapshot: Snapshot{Account: renamed.Account, Name: old.Name, LeftAt: at}}, true
}

func (p PastName) TakeSnapshot() Snapshot { return p.snapshot }

// A name left as the event told it (player.v1.NameLeft): one with no name or no time is refused, so a broken event is
// buried rather than kept.
func (s Snapshot) Restore() (PastName, error) {
	if s.Name == "" {
		return PastName{}, ErrNoName
	}
	if s.LeftAt.IsZero() {
		return PastName{}, ErrNoTime
	}
	return PastName{snapshot: s}, nil
}

// The store only writes: what it keeps is read by the public card's query (player_query).
type Storage interface {
	// Keeps a name once: the same account's name left at the same time again changes nothing, so an event heard twice
	// keeps one.
	Record(ctx context.Context, name PastName) error
	Forget(ctx context.Context, account players.AccountID) error
}
