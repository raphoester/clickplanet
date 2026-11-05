package forget_link_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Store interface {
	LinkOf(ctx context.Context, account links.AccountID) (links.Link, error)
	SaveLinks(ctx context.Context, links ...links.Link) error
}

func New(store Store) *UseCase {
	return &UseCase{store: store}
}

type UseCase struct {
	store Store
}

// The Discord user stays in the server with its roles: that is its own Discord account, not the game's data.
func (u *UseCase) Execute(ctx context.Context, account links.AccountID) error {
	link, err := u.store.LinkOf(ctx, account)
	if errors.Is(err, links.ErrNoLink) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the link: %w", err)
	}
	link.Forget()
	if err := u.store.SaveLinks(ctx, link); err != nil {
		return fmt.Errorf("failed to forget the link: %w", err)
	}
	return nil
}
