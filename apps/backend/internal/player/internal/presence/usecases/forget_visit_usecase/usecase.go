package forget_visit_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Visits interface {
	Forget(account players.AccountID)
}

type UseCase struct {
	visits Visits
}

func New(visits Visits) *UseCase {
	return &UseCase{visits: visits}
}

func (u *UseCase) Execute(_ context.Context, account players.AccountID) error {
	u.visits.Forget(account)
	return nil
}
