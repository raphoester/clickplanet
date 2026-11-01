package podiums

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store interface {
	Record(ctx context.Context, places []Place) error
	Places(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]Places, error)
	DeleteAccount(ctx context.Context, account players.AccountID) error
}
