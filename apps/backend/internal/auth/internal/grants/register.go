package grants

import (
	"context"
	"fmt"
	"time"
)

type Saver interface {
	SaveGrant(ctx context.Context, grant Grant) error
}

func NewRegister(store Saver, ids IDProvider) *Register {
	return &Register{store: store, ids: ids}
}

// Where a Discord sign-in is saved, the leave it gave to join the server is kept beside it.
type Register struct {
	store Saver
	ids   IDProvider
}

// A grant with an id of its own, kept nowhere yet.
func (r *Register) NewGrant(account AccountID, user string, token string, now time.Time) (Grant, error) {
	id, err := r.ids.NewID()
	if err != nil {
		return Grant{}, fmt.Errorf("failed to get a grant id: %w", err)
	}
	return NewGrant(id, account, user, token, now)
}

func (r *Register) Keep(ctx context.Context, grant Grant) error {
	if err := r.store.SaveGrant(ctx, grant); err != nil {
		return fmt.Errorf("failed to keep the grant: %w", err)
	}
	return nil
}
