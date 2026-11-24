package submit_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Gallery interface {
	Redrawn(ctx context.Context, id pictures.UploadID) (pictures.Drawn, error)
	Hold(ctx context.Context, key string, drawn pictures.Drawn) error
	Discard(ctx context.Context, id pictures.UploadID) error
}

type Submissions interface {
	Founded(ctx context.Context, account submissions.AccountID) ([]submissions.Submission, error)
	Save(ctx context.Context, submissions ...submissions.Submission) error
}

type In struct {
	Account   string
	Linked    bool
	Name      string
	Upload    string
	Latitude  float64
	Longitude float64
	// Empty for a new flag.
	Flag string
}

const timeout = 30 * time.Second

func New(ids submissions.IDProvider, clock cptime.Clock, gallery Gallery, store Submissions) *UseCase {
	return &UseCase{ids: ids, clock: clock, gallery: gallery, store: store}
}

type UseCase struct {
	ids     submissions.IDProvider
	clock   cptime.Clock
	gallery Gallery
	store   Submissions
}

func (u *UseCase) Execute(ctx context.Context, in In) (submissions.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	account, err := submissions.FounderOf(in.Account, in.Linked)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}
	draft, err := submissions.ParseDraft(in.Name, in.Latitude, in.Longitude, in.Flag)
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
	submission, err := submissions.Submit(id, account, draft, drawn, earlier, u.clock.Now())
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}

	if err := u.gallery.Hold(ctx, id.PictureKey(), drawn); err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the gallery names it.
	}
	if err := u.store.Save(ctx, submission); err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to keep the submission: %w", err)
	}
	// The picture is held: a file left in the private folder harms nobody.
	_ = u.gallery.Discard(ctx, upload)
	return submission.TakeSnapshot(), nil
}
