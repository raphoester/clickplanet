package grants

import "context"

type Store interface {
	// The payment's grant, Unheard when none was kept, locked to the end of the transaction a change reads it in.
	Grant(ctx context.Context, id GrantID) (Grant, error)
	Save(ctx context.Context, grant Grant) error
}
