package titles

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Store interface {
	Held(ctx context.Context, account accounts.AccountID) (IDs, error)
	Holdings(ctx context.Context, accounts []accounts.AccountID) (Holdings, error)
	Grant(ctx context.Context, grants Holdings, at time.Time) error
	Revoke(ctx context.Context, revocations Holdings) error
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}
