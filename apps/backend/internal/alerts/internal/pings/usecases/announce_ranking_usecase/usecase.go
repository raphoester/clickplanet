package announce_ranking_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	RankingPinged(ctx context.Context, round pings.RoundID) (bool, error)
	SaveRankingPing(ctx context.Context, round pings.RoundID, at time.Time) error
}

type Results interface {
	RankingAfter(ctx context.Context, round pings.RoundID) (pings.Ranking, error)
}

type Herald interface {
	TellRanking(ctx context.Context, day pings.EndedDay, ranking pings.Ranking) error
}

const timeout = 10 * time.Second

func New(
	clock cptime.Clock, store Store, results Results, herald Herald, transactioner cpatomic.Transactioner,
) *UseCase {
	return &UseCase{clock: clock, store: store, results: results, herald: herald, transactioner: transactioner}
}

type UseCase struct {
	clock         cptime.Clock
	store         Store
	results       Results
	herald        Herald
	transactioner cpatomic.Transactioner
}

// Told once: a post Discord refused, or a round scoring could not read, is kept for nobody, and the event comes again
// until it is RankingFresh no more. A day nobody scored in tells nothing.
func (u *UseCase) Execute(ctx context.Context, day pings.EndedDay) error {
	now := u.clock.Now()
	if day.Stale(now) {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pinged, err := u.store.RankingPinged(ctx, day.Round)
	if err != nil {
		return fmt.Errorf("failed to read whether the ranking of %s was told: %w", day.Round, err)
	}
	if pinged {
		return nil
	}
	ranking, err := u.results.RankingAfter(ctx, day.Round)
	if err != nil {
		return fmt.Errorf("failed to read what %s scored: %w", day.Round, err)
	}
	if ranking.Empty() {
		return nil
	}
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step named its own.
		if err := u.store.SaveRankingPing(ctx, day.Round, now); err != nil {
			return fmt.Errorf("failed to keep the ping: %w", err)
		}
		return u.herald.TellRanking(ctx, day, ranking)
	})
}
