package approve_submission_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Gallery interface {
	Release(ctx context.Context, key string, picture pictures.Picture) error
	Drop(ctx context.Context, key string) error
}

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Flags(ctx context.Context) ([]catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

const timeout = 30 * time.Second

func New(
	clock cptime.Clock, gallery Gallery, db cpatomic.Transactioner, store submissions.Store, flags Flags,
	outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{clock: clock, gallery: gallery, db: db, store: store, flags: flags, outbox: outbox}
}

type UseCase struct {
	clock   cptime.Clock
	gallery Gallery
	db      cpatomic.Transactioner
	store   submissions.Store
	flags   Flags
	outbox  cpatomic.Outbox
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
	now := u.clock.Now()
	approved, err := waiting.Approved(catalog.NamesOf(flags), now)
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

	// Never a change approved that its flag does not wear, nor a flag changed that the pages are not told of.
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		if err := u.store.Save(ctx, approved); err != nil {
			return fmt.Errorf("failed to keep the approval: %w", err)
		}
		if kept.Kind != submissions.Change {
			return nil
		}
		flag, err := u.flags.Flag(ctx, kept.Flag)
		if err != nil {
			return fmt.Errorf("failed to read the flag changed: %w", err)
		}
		restyled, facts, err := approved.Restyle(flag, now)
		if err != nil {
			return err //nolint:wrapcheck // the rule names it.
		}
		if err := u.flags.Save(ctx, restyled); err != nil {
			return fmt.Errorf("failed to keep the flag changed: %w", err)
		}
		if err := u.outbox.Record(ctx, facts...); err != nil {
			return fmt.Errorf("failed to record that the flag changed: %w", err)
		}
		return nil
	}); err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to approve the submission %s: %w", id, err)
	}
	// The picture is public: the private copy is no longer read.
	_ = u.gallery.Drop(ctx, id.PictureKey())
	return kept, nil
}
