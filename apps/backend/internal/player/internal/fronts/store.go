package fronts

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store interface {
	RecordTake(ctx context.Context, take Take) error
	DeleteAccount(ctx context.Context, account players.AccountID) error
}
