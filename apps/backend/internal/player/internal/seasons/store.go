package seasons

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store interface {
	Record(ctx context.Context, account players.AccountID) (Record, error)
	Records(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]Record, error)
	Save(ctx context.Context, record Record) error
}
