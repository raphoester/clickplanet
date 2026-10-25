package rules

import "context"

type Store interface {
	// Change is the change kept for the request, or none.
	Change(ctx context.Context, request RequestID) (Change, error)
	// Latest is the change applied last, or none.
	Latest(ctx context.Context) (Change, error)
	Save(ctx context.Context, change Change) error
}
