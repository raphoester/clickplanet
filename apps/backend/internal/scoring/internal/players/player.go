package players

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

// An account player made a player, as player tells it (PlayerCreated): it has a username. Every other account is a
// guest, which no board ranks.
type Store interface {
	Keep(ctx context.Context, account AccountID) error
	IsPlayer(ctx context.Context, account AccountID) (bool, error)
	Forget(ctx context.Context, account AccountID) error
}
