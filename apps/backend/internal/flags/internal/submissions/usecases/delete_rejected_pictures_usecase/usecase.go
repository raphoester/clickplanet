package delete_rejected_pictures_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Gallery interface {
	Drop(ctx context.Context, key string) error
}

func New(clock cptime.Clock, store submissions.Store, gallery Gallery) *UseCase {
	return &UseCase{clock: clock, store: store, gallery: gallery}
}

// UseCase deletes each rejected picture past its time. It never became public.
type UseCase struct {
	clock   cptime.Clock
	store   submissions.Store
	gallery Gallery
}

func (u *UseCase) Execute(ctx context.Context) error {
	rejected, err := u.store.RejectedHeld(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the rejected submissions: %w", err)
	}
	now := u.clock.Now()
	var errs []error
	for _, submission := range rejected {
		deleted, due := submission.PictureDeleted(now)
		if !due {
			continue
		}
		id := deleted.TakeSnapshot().ID
		// The file goes first: a submission that says so holds none.
		if err := u.gallery.Drop(ctx, id.PictureKey()); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := u.store.Save(ctx, deleted); err != nil {
			errs = append(errs, fmt.Errorf("failed to keep that the picture of %s is deleted: %w", id, err))
		}
	}
	return errors.Join(errs...)
}
