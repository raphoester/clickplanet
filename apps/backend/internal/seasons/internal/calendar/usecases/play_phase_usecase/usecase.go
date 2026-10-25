package play_phase_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Calendar interface {
	PhaseAt(now time.Time) (calendar.Phase, bool)
}

type Game interface {
	Play(ctx context.Context, phase calendar.Phase) error
}

type Executor interface {
	// Execute answers the phase it told the game to play, and none when the phase did not change.
	Execute(ctx context.Context) (calendar.Phase, error)
}

func New(seasons Calendar, game Game, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, game: game, clock: clock}
}

// UseCase tells the game the phase once it changes, and again after each boot: what the game kept may be stale.
type UseCase struct {
	seasons Calendar
	game    Game
	clock   cptime.Clock

	// TODO(broker): in memory, so each boot sends the phase again; a broker delivers a command once.
	played calendar.Phase
}

func (u *UseCase) Execute(ctx context.Context) (calendar.Phase, error) {
	phase, ok := u.seasons.PhaseAt(u.clock.Now())
	if !ok || phase == u.played {
		return calendar.Phase{}, nil
	}

	if err := u.game.Play(ctx, phase); err != nil {
		return calendar.Phase{}, err //nolint:wrapcheck // the adapter says what failed.
	}
	u.played = phase

	return phase, nil
}
