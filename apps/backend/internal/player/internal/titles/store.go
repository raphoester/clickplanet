package titles

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store interface {
	Held(ctx context.Context, account players.AccountID) (IDs, error)
	Grant(ctx context.Context, grants Grants, at time.Time) error
	DeleteAccount(ctx context.Context, account players.AccountID) error
}
