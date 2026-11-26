package submissions

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

type Store interface {
	// ErrUnknownSubmission for one never kept.
	Submission(ctx context.Context, id SubmissionID) (Submission, error)
	// Every submission of the account, the newest first.
	Founded(ctx context.Context, account catalog.AccountID) ([]Submission, error)
	// The submission the flag was opened from: none for a flag opened by hand.
	Opened(ctx context.Context, flag catalog.FlagID) ([]Submission, error)
	// Every rejected submission whose picture the bucket still holds.
	RejectedHeld(ctx context.Context) ([]Submission, error)
	Save(ctx context.Context, submissions ...Submission) error
	// Deletes a submission that still waits: ErrNotWaiting for one the operator decided, or never kept.
	DeleteWaiting(ctx context.Context, id SubmissionID) error
}
