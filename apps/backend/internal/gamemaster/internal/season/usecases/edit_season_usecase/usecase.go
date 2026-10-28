package edit_season_usecase

import (
	"context"
	"fmt"
	"time"

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

func New(seasons Seasons, entries Entries, keepers []schedule.Keeper, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, entries: entries, keepers: keepers, clock: clock}
}

// UseCase changes a season that has not ended. Scoring hears of a running one's new end at the next play of the seasons.
// Its finale names a set every module keeps, and no battle overlaps any finale once the seasons after it moved.
type UseCase struct {
	seasons Seasons
	entries Entries
	keepers []schedule.Keeper
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, number season.Number, draft season.Draft) error {
	now := u.clock.Now()
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return fmt.Errorf("could not read the seasons: %w", err)
	}
	edited, err := calendar.Edited(number, draft, now)
	if err != nil {
		return err //nolint:wrapcheck // the calendar says why.
	}
	if draft.Finale.Planned() {
		if err := schedule.Playable(ctx, draft.Finale.Set, u.keepers...); err != nil {
			return fmt.Errorf("cannot schedule the finale's rule set %q: %w", draft.Finale.Set, err)
		}
	}
	entries, err := u.entries.Unended(ctx, now)
	if err != nil {
		return fmt.Errorf("could not read the schedule: %w", err)
	}
	if err := calendar.With(edited).Overlapped(entries...); err != nil {
		return err //nolint:wrapcheck // the calendar names each battle.
	}
	if err := u.seasons.Save(ctx, edited); err != nil {
		return fmt.Errorf("could not keep season %d: %w", number, err)
	}
	return nil
}
