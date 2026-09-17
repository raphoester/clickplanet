package players

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNoProfile is an account that never chose a name.
	ErrNoProfile = errors.New("the account has no profile")
	// ErrNoStats is an account that never took a tile.
	ErrNoStats = errors.New("the account has no stats")
	// ErrNameTaken is a name another account holds, ignoring case.
	ErrNameTaken = errors.New("another player has this name")
)

// Store is where profiles and stats are kept. Every call reads or writes the database.
type Store interface {
	// Profile answers ErrNoProfile when the account never chose a name.
	Profile(ctx context.Context, account AccountID) (Profile, error)
	// ProfileNamed is the profile holding name, ignoring case. ErrNoProfile when no account holds it.
	ProfileNamed(ctx context.Context, name Name) (Profile, error)
	// SaveProfile writes the profile over the account's last one. ErrNameTaken when another account holds the
	// name ignoring case; the account's own name, in any case, is not taken.
	SaveProfile(ctx context.Context, profile Profile) error
	// Stats answers ErrNoStats when the account never took a tile.
	Stats(ctx context.Context, account AccountID) (Stats, error)
	// RecordTake counts one more tile taken at at, by Stats.WithTake, as one atomic read and write.
	RecordTake(ctx context.Context, account AccountID, at time.Time) error
	// DeleteAccount deletes the account's profile and stats. An account with neither is not an error.
	DeleteAccount(ctx context.Context, account AccountID) error
	// Names is the name of every account given that has one.
	Names(ctx context.Context, accounts []AccountID) (map[AccountID]Name, error)
}
