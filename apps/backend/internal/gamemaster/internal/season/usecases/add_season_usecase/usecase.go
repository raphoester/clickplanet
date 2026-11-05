package add_season_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
	Save(ctx context.Context, season season.Season) error
}

type Entries interface {
	Unended(ctx context.Context, at time.Time) ([]schedule.Entry, error)
}

func New(seasons Seasons, entries Entries, keepers []schedule.Keeper, cutoff season.Cutoff, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, entries: entries, keepers: keepers, cutoff: cutoff, clock: clock}
}

// UseCase schedules a season to come, under the next number. Its finale names a set every module keeps, and every
// battle fits between its days.
type UseCase struct {
	seasons Seasons
	entries Entries
	keepers []schedule.Keeper
	cutoff  season.Cutoff
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, draft season.Draft) (season.Number, error) {
	now := u.clock.Now()
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not read the seasons: %w", err)
	}
	added, err := calendar.Added(draft, now)
	if err != nil {
		return 0, err //nolint:wrapcheck // the calendar says why.
	}
	if draft.Finale.Planned() {
		if err := schedule.Playable(ctx, draft.Finale.Set, u.keepers...); err != nil {
			return 0, fmt.Errorf("cannot schedule the finale's rule set %q: %w", draft.Finale.Set, err)
		}
	}
	entries, err := u.entries.Unended(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("could not read the schedule: %w", err)
	}
	if err := moments.NewPlan(calendar.With(added), u.cutoff, nil, entries...).Clashes(now); err != nil {
		return 0, err //nolint:wrapcheck // the plan names each battle.
	}
	if err := u.seasons.Save(ctx, added); err != nil {
		return 0, fmt.Errorf("could not keep the season: %w", err)
	}
	return added.TakeSnapshot().Number, nil
}
