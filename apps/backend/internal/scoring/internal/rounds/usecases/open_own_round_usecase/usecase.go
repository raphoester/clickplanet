package open_own_round_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Rounds interface {
	Open(ctx context.Context, round rounds.Round) error
}

// Waker is the runner that closes the rounds: a round of its own may end before the one it waits for.
type Waker interface {
	Wake()
}

func New(seasons calendar.Calendar, store Rounds, clock cptime.Clock, waker Waker) *UseCase {
	return &UseCase{seasons: seasons, store: store, clock: clock, waker: waker}
}

type UseCase struct {
	seasons calendar.Calendar
	store   Rounds
	clock   cptime.Clock
	waker   Waker
}

// Execute opens the round of its own of an entry's window. Opened twice, it is one round; heard of once it ended, it is
// none.
func (u *UseCase) Execute(ctx context.Context, times uint32, startsAt, endsAt time.Time) error {
	round, ok := rounds.Own(u.seasons, times, startsAt, endsAt, u.clock.Now())
	if !ok {
		return nil
	}
	if err := u.store.Open(ctx, round); err != nil {
		return fmt.Errorf("failed to open the round of its own: %w", err)
	}
	u.waker.Wake()
	return nil
}
