package add_entry_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Save(ctx context.Context, entry schedule.Entry) error
}

func New(entries Entries, keepers []schedule.Keeper, ids schedule.IDProvider, clock cptime.Clock) *UseCase {
	return &UseCase{entries: entries, keepers: keepers, ids: ids, clock: clock}
}

// UseCase schedules an entry to come, of a set every module keeps.
type UseCase struct {
	entries Entries
	keepers []schedule.Keeper
	ids     schedule.IDProvider
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, draft schedule.Draft) (schedule.EntryID, error) {
	id, err := u.ids.NewID()
	if err != nil {
		return schedule.EntryID{}, fmt.Errorf("could not name the entry: %w", err)
	}
	entry, err := schedule.NewEntry(id, draft, u.clock.Now())
	if err != nil {
		return schedule.EntryID{}, err //nolint:wrapcheck // the entry says every reason.
	}
	if err := schedule.Playable(ctx, draft.Set, u.keepers...); err != nil {
		return schedule.EntryID{}, fmt.Errorf("cannot schedule the rule set %q: %w", draft.Set, err)
	}
	if err := u.entries.Save(ctx, entry); err != nil {
		return schedule.EntryID{}, fmt.Errorf("could not keep the entry: %w", err)
	}
	return id, nil
}
