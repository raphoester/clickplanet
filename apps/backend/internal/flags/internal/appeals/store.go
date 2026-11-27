package appeals

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Store interface {
	// The submission's appeals: one at most.
	Of(ctx context.Context, submission submissions.SubmissionID) ([]Appeal, error)
	Save(ctx context.Context, appeal Appeal) error
}
