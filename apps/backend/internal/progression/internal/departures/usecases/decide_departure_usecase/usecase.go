package decide_departure_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
)

type Departures interface {
	Change(ctx context.Context, account accounts.AccountID, change func(departures.Departure) departures.Departure) error
}

type UseCase struct {
	departures Departures
}

func New(departures Departures) *UseCase {
	return &UseCase{departures: departures}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID, decision departures.Decision) error {
	if err := decision.TimeError(); err != nil {
		return fmt.Errorf("failed to read the decision on %s: %w", account, err)
	}
	if err := u.departures.Change(ctx, account, func(departure departures.Departure) departures.Departure {
		return departure.Decided(decision)
	}); err != nil {
		return fmt.Errorf("failed to keep the decision on %s: %w", account, err)
	}
	return nil
}
