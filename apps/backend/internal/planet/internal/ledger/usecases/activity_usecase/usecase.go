package activity_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const defaultSpan = 24 * time.Hour

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type Grid interface {
	Land() []byte
	Heat(changes map[uint32]int) []uint32
}

type Territories interface {
	Territories() map[string]uint32
}

type Audience interface {
	Audience() bonuses.Audience
}

type In struct {
	Span time.Duration
}

type Out struct {
	Activity ledger.Summary
	Land     []byte
	Heat     []uint32
	Held     map[string]uint32
	Audience bonuses.Audience
}

func New(ledger Ledger, grid Grid, territories Territories, audience Audience, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, grid: grid, territories: territories, audience: audience, clock: clock}
}

type UseCase struct {
	ledger      Ledger
	grid        Grid
	territories Territories
	audience    Audience
	clock       cptime.Clock
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	span := in.Span
	if span == 0 {
		span = defaultSpan
	}

	activity, err := ledger.NewActivity(u.clock.Now(), span)
	if err != nil {
		return Out{}, fmt.Errorf("failed to cut %s into slots: %w", span, err)
	}
	u.ledger.Replay(activity.See)

	summary := activity.Summary()

	return Out{
		Activity: summary,
		Land:     u.grid.Land(),
		Heat:     u.grid.Heat(summary.Changes),
		Held:     u.territories.Territories(),
		Audience: u.audience.Audience(),
	}, nil
}
