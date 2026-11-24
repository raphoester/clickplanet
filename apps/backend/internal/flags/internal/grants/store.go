package grants

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Store interface {
	// The payment's grant, Unheard when none was kept, locked to the end of the transaction a change reads it in.
	Grant(ctx context.Context, id GrantID) (Grant, error)
	// The grants held for the submission and neither released nor taken back, the first paid first, each locked.
	Held(ctx context.Context, submission submissions.SubmissionID) ([]Grant, error)
	Save(ctx context.Context, grant Grant) error
}
