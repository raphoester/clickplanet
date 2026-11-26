package submit_flag_usecase

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
	Redrawn(ctx context.Context, id pictures.UploadID) (pictures.Drawn, error)
	Hold(ctx context.Context, key string, drawn pictures.Drawn) error
	Discard(ctx context.Context, id pictures.UploadID) error
	Drop(ctx context.Context, key string) error
}

type Submissions interface {
	Founded(ctx context.Context, account catalog.AccountID) ([]submissions.Submission, error)
	Save(ctx context.Context, submissions ...submissions.Submission) error
	DeleteWaiting(ctx context.Context, id submissions.SubmissionID) error
}

type In struct {
	Account   string
	Linked    bool
	Name      string
	Upload    string
	Latitude  float64
	Longitude float64
	// The account's submission that waits, which this one replaces; empty for none.
	Replaces string
}

const timeout = 30 * time.Second

func New(
	ids submissions.IDProvider, clock cptime.Clock, gallery Gallery, db cpatomic.Transactioner, store Submissions,
) *UseCase {
	return &UseCase{ids: ids, clock: clock, gallery: gallery, db: db, store: store}
}

type UseCase struct {
	ids     submissions.IDProvider
	clock   cptime.Clock
	gallery Gallery
	db      cpatomic.Transactioner
	store   Submissions
}

func (u *UseCase) Execute(ctx context.Context, in In) (submissions.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	account, err := submissions.FounderOf(in.Account, in.Linked)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}
	draft, err := submissions.ParseDraft(in.Name, in.Latitude, in.Longitude, in.Replaces)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	upload, err := pictures.ParseUploadID(in.Upload)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	earlier, err := u.store.Founded(ctx, account)
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to read the account's submissions: %w", err)
	}
	drawn, err := u.gallery.Redrawn(ctx, upload)
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to draw the picture again: %w", err)
	}
	id, err := u.ids.NewID()
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to draw a submission id: %w", err)
	}
	submission, replaced, err := submissions.Submit(id, account, draft, drawn, earlier, u.clock.Now())
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}

	if err := u.gallery.Hold(ctx, id.PictureKey(), drawn); err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the gallery names it.
	}
	// Never two that wait, nor the one that waited lost with nothing in its place.
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		for _, old := range replaced {
			if err := u.store.DeleteWaiting(ctx, old); err != nil {
				return fmt.Errorf("failed to delete the submission replaced: %w", err)
			}
		}
		if err := u.store.Save(ctx, submission); err != nil {
			return fmt.Errorf("failed to keep the submission: %w", err)
		}
		return nil
	}); err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the transaction names it.
	}
	// The pictures are held or gone: a file left in the private folder harms nobody.
	_ = u.gallery.Discard(ctx, upload)
	for _, old := range replaced {
		_ = u.gallery.Drop(ctx, old.PictureKey())
	}
	return submission.TakeSnapshot(), nil
}
