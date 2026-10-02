package players

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNoProfile = errors.New("the account has no profile")
	ErrNoStats   = errors.New("the account has no stats")
	ErrNameTaken = errors.New("another player has this name")
)

type Store interface {
	GuestCodeStore

	Profile(ctx context.Context, account AccountID) (Profile, error)
	ProfileNamed(ctx context.Context, name Name) (Profile, error)
	SaveProfile(ctx context.Context, profile Profile) error
	SaveColor(ctx context.Context, account AccountID, color Color) error
	Stats(ctx context.Context, account AccountID) (Stats, error)
	RecordTake(ctx context.Context, account AccountID, at time.Time) error
	StatsAfter(ctx context.Context, after AccountID, limit int) ([]Stats, error)
	DeleteAccount(ctx context.Context, account AccountID) error
	Names(ctx context.Context, accounts []AccountID) (map[AccountID]Name, error)
	Authors(ctx context.Context, accounts []AccountID) (map[AccountID]Author, error)
}
