package finales

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store interface {
	Record(ctx context.Context, fights []Fight) error
	Fights(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]Fights, error)
	DeleteAccount(ctx context.Context, account players.AccountID) error
}
