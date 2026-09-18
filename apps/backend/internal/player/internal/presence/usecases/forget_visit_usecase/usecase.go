// Package forget_visit_usecase takes an account off the roster at once: it signed out, or it is gone.
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

// Execute never fails. It answers an error to fit the subscribers, which also serve use cases that write postgres.
func (u *UseCase) Execute(_ context.Context, account players.AccountID) error {
	u.visits.Forget(account)
	return nil
}
