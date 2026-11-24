package approve_submission_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Gallery interface {
	Release(ctx context.Context, key string, picture pictures.Picture) error
	Drop(ctx context.Context, key string) error
}

type Flags interface {
	Flags(ctx context.Context) ([]catalog.Flag, error)
}

const timeout = 30 * time.Second

func New(clock cptime.Clock, gallery Gallery, store submissions.Store, flags Flags) *UseCase {
	return &UseCase{clock: clock, gallery: gallery, store: store, flags: flags}
}

type UseCase struct {
	clock   cptime.Clock
	gallery Gallery
	store   submissions.Store
	flags   Flags
}

type In struct {
	Submission string
	// The operator moved its capital before the verdict.
	Moved     bool
	Latitude  float64
	Longitude float64
}

func (u *UseCase) Execute(ctx context.Context, in In) (submissions.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := submissions.ParseSubmissionID(in.Submission)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	waiting, err := u.store.Submission(ctx, id)
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	if in.Moved {
		pin, err := catalog.ParsePin(in.Latitude, in.Longitude)
		if err != nil {
			return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
		}
		if waiting, err = waiting.Moved(pin); err != nil {
			return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
		}
	}
	flags, err := u.flags.Flags(ctx)
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to read the flags' names: %w", err)
	}
	approved, err := waiting.Approved(catalog.NamesOf(flags), u.clock.Now())
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}
	kept := approved.TakeSnapshot()
	picture, err := kept.Picture.Restore()
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("the submission %s: %w", id, err)
	}
	if err := u.gallery.Release(ctx, id.PictureKey(), picture); err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the gallery names it.
	}
	if err := u.store.Save(ctx, approved); err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to keep the approval of %s: %w", id, err)
	}
	// The picture is public: the private copy is no longer read.
	_ = u.gallery.Drop(ctx, id.PictureKey())
	return kept, nil
}
