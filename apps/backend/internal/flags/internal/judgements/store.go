package judgements

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Store interface {
	// The submissions that wait with no judgement, the first sent first, at most limit.
	Unjudged(ctx context.Context, limit int) ([]submissions.SubmissionID, error)
	// ErrUnjudged for a submission the judge never judged.
	Judgement(ctx context.Context, submission submissions.SubmissionID) (Judgement, error)
	Save(ctx context.Context, judgement Judgement) error
}
