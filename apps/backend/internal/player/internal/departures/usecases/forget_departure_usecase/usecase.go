package forget_departure_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Departures interface {
	Forget(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	departures Departures
}

func New(departures Departures) *UseCase {
	return &UseCase{departures: departures}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.departures.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the departure: %w", err)
	}
	return nil
}
