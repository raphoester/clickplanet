package edit_entry_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Entry(ctx context.Context, id schedule.EntryID) (schedule.Entry, error)
	Save(ctx context.Context, entry schedule.Entry) error
}

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

func New(entries Entries, seasons Seasons, keepers []schedule.Keeper, clock cptime.Clock) *UseCase {
	return &UseCase{entries: entries, seasons: seasons, keepers: keepers, clock: clock}
}

// UseCase changes an entry that has not started, by the rules of a new one.
type UseCase struct {
	entries Entries
	seasons Seasons
	keepers []schedule.Keeper
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, id schedule.EntryID, draft schedule.Draft) error {
	entry, err := u.entries.Entry(ctx, id)
	if err != nil {
		return err //nolint:wrapcheck // the store names the entry.
	}
	edited, err := entry.Edited(draft, u.clock.Now())
	if err != nil {
		return err //nolint:wrapcheck // the entry says every reason.
	}
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return fmt.Errorf("could not read the seasons: %w", err)
	}
	if err := calendar.Overlapped(edited); err != nil {
		return err //nolint:wrapcheck // the calendar names the finale.
	}
	if err := schedule.Playable(ctx, draft.Set, u.keepers...); err != nil {
		return fmt.Errorf("cannot schedule the rule set %q: %w", draft.Set, err)
	}
	if err := u.entries.Save(ctx, edited); err != nil {
		return fmt.Errorf("could not keep the entry %s: %w", id, err)
	}
	return nil
}
