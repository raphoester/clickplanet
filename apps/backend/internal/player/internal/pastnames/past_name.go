// Package pastnames keeps every username a player chose to leave for another: the ledger its public card reads its
// last names from (GetPlayer), so the others can tell who a player was. Kept since 2026-11-17, until the account is
// forgotten. A name an operator replaced is not one: that is the renames' history, the operator's alone, and a name
// taken off for its words must not come back on the card.
package pastnames

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
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

// The store only writes: what it keeps is read by the public card's query (player_query).
type Storage interface {
	Record(ctx context.Context, name PastName) error
	Forget(ctx context.Context, account players.AccountID) error
}
