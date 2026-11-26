package judgements

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Submissions interface {
	Submission(ctx context.Context, id submissions.SubmissionID) (submissions.Submission, error)
}

type Pictures interface {
	Held(ctx context.Context, key string, picture pictures.Picture) ([]byte, error)
}

type Flags interface {
	Flags(ctx context.Context) ([]catalog.Flag, error)
}

// Cases reads what the judge sees of a submission: the submission, its picture held in private, and the names of the
// flags in the game, open or closed at their end. A removed flag's place is free for a new flag.
type Cases struct {
	Submissions Submissions
	Pictures    Pictures
	Flags       Flags
}

func (c Cases) Case(ctx context.Context, id submissions.SubmissionID) (Case, error) {
	submission, err := c.Submissions.Submission(ctx, id)
	if err != nil {
		return Case{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	if !submission.Waiting() {
		return Case{}, fmt.Errorf("%w: %s", ErrNotWaiting, id)
	}
	kept := submission.TakeSnapshot()
	picture, err := kept.Picture.Restore()
	if err != nil {
		return Case{}, fmt.Errorf("the submission %s: %w", id, err)
	}
	png, err := c.Pictures.Held(ctx, id.PictureKey(), picture)
	if err != nil {
		return Case{}, err //nolint:wrapcheck // the gallery names it.
	}
	flags, err := c.Flags.Flags(ctx)
	if err != nil {
		return Case{}, fmt.Errorf("failed to read the flags' names: %w", err)
	}
	taken := make([]catalog.Name, 0, len(flags))
	for _, flag := range flags {
		if kept := flag.TakeSnapshot(); kept.State != catalog.Removed {
			taken = append(taken, kept.Name)
		}
	}
	return Case{Name: kept.Name, Pin: kept.Pin, Picture: png, Taken: taken}, nil
}
