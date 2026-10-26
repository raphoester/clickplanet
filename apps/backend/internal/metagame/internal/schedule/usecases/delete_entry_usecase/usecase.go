package delete_entry_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Entries interface {
	Entry(ctx context.Context, id schedule.EntryID) (schedule.Entry, error)
	Delete(ctx context.Context, id schedule.EntryID) error
}

func New(entries Entries, clock cptime.Clock) *UseCase {
	return &UseCase{entries: entries, clock: clock}
}

// UseCase deletes an entry that has not started.
type UseCase struct {
	entries Entries
	clock   cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, id schedule.EntryID) error {
	entry, err := u.entries.Entry(ctx, id)
	if err != nil {
		return err //nolint:wrapcheck // the store names the entry.
	}
	if err := entry.Changeable(u.clock.Now()); err != nil {
		return err //nolint:wrapcheck // the entry says when it started.
	}
	if err := u.entries.Delete(ctx, id); err != nil {
		return fmt.Errorf("could not delete the entry %s: %w", id, err)
	}
	return nil
}
