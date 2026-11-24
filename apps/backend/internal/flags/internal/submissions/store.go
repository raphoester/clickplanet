package submissions

import "context"

type Store interface {
	// ErrUnknownSubmission for one never kept.
	Submission(ctx context.Context, id SubmissionID) (Submission, error)
	// Every submission of the account, the newest first.
	Founded(ctx context.Context, account AccountID) ([]Submission, error)
	// Every rejected submission whose picture the bucket still holds.
	RejectedHeld(ctx context.Context) ([]Submission, error)
	Save(ctx context.Context, submissions ...Submission) error
}
