package finales

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Store interface {
	Record(ctx context.Context, fights []Fight) error
	Fights(ctx context.Context, accounts []accounts.AccountID) (map[accounts.AccountID]Fights, error)
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}
