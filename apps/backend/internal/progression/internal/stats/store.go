package stats

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

var ErrNoStats = errors.New("the account has no stats")

type Store interface {
	Stats(ctx context.Context, account accounts.AccountID) (Stats, error)
	Record(ctx context.Context, account accounts.AccountID, change func(Stats) Stats) error
	StatsAfter(ctx context.Context, after accounts.AccountID, limit int) ([]Stats, error)
	Forget(ctx context.Context, account accounts.AccountID) error
}
