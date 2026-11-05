package add_entry_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Save(ctx context.Context, entry schedule.Entry) error
}

type Plans interface {
	With(ctx context.Context, changed ...schedule.Entry) (moments.Plan, error)
}

func New(entries Entries, plans Plans, keepers []schedule.Keeper, ids schedule.IDProvider, clock cptime.Clock) *UseCase {
	return &UseCase{entries: entries, plans: plans, keepers: keepers, ids: ids, clock: clock}
}

// UseCase schedules an entry to come, of a set every module keeps. A battle fits between the season's days.
type UseCase struct {
	entries Entries
	plans   Plans
	keepers []schedule.Keeper
	ids     schedule.IDProvider
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, draft schedule.Draft) (schedule.EntryID, error) {
	id, err := u.ids.NewID()
	if err != nil {
		return schedule.EntryID{}, fmt.Errorf("could not name the entry: %w", err)
	}
	now := u.clock.Now()
	entry, err := schedule.NewEntry(id, draft, now)
	if err != nil {
		return schedule.EntryID{}, err //nolint:wrapcheck // the entry says every reason.
	}
	plan, err := u.plans.With(ctx, entry)
	if err != nil {
		return schedule.EntryID{}, fmt.Errorf("could not plan the seasons: %w", err)
	}
	if err := plan.Clashes(now); err != nil {
		return schedule.EntryID{}, err //nolint:wrapcheck // the plan names each battle.
	}
	if err := schedule.Playable(ctx, draft.Set, u.keepers...); err != nil {
		return schedule.EntryID{}, fmt.Errorf("cannot schedule the rule set %q: %w", draft.Set, err)
	}
	if err := u.entries.Save(ctx, entry); err != nil {
		return schedule.EntryID{}, fmt.Errorf("could not keep the entry: %w", err)
	}
	return id, nil
}
