package nameplates

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

// An account with a username, as player tells it: the other players see the progression of no other.
type Store interface {
	Keep(ctx context.Context, account accounts.AccountID) error
	Forget(ctx context.Context, account accounts.AccountID) error
}
