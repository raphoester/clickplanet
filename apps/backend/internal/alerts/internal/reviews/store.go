package reviews

import (
	"context"
	"time"
)

type Store interface {
	// Whether the staff were told of the submission for the cause.
	Told(ctx context.Context, submission SubmissionID, cause Cause) (bool, error)
	SaveTold(ctx context.Context, submission SubmissionID, cause Cause, at time.Time) error
}
