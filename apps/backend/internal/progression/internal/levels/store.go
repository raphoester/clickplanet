package levels

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Store interface {
	Levels(ctx context.Context, accounts []accounts.AccountID) (ByAccount, error)
	// Raise answers the level kept before it, 0 for none, and writes only above it.
	Raise(ctx context.Context, account accounts.AccountID, level Level, at time.Time) (Level, error)
	RaiseAll(ctx context.Context, levels ByAccount, at time.Time) error
	Set(ctx context.Context, levels ByAccount, at time.Time) error
	Forget(ctx context.Context, accounts ...accounts.AccountID) error
}
