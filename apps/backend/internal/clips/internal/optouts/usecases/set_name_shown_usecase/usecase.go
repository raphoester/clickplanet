package set_name_shown_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
)

type Store interface {
	Keep(ctx context.Context, account optouts.AccountID) error
	Forget(ctx context.Context, account optouts.AccountID) error
}

func New(store Store) *UseCase {
	return &UseCase{store: store}
}

type UseCase struct {
	store Store
}

// The runs after this one name the player or not; a cast already made keeps its names.
func (u *UseCase) Execute(ctx context.Context, account optouts.AccountID, shown bool) error {
	if shown {
		if err := u.store.Forget(ctx, account); err != nil {
			return fmt.Errorf("failed to let the clips name the player again: %w", err)
		}
		return nil
	}
	if err := u.store.Keep(ctx, account); err != nil {
		return fmt.Errorf("failed to keep the player's name out of the clips: %w", err)
	}
	return nil
}
