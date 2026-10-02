package set_color_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Colors interface {
	SaveColor(ctx context.Context, account players.AccountID, color players.Color) error
}

type UseCase struct {
	colors Colors
}

func New(colors Colors) *UseCase {
	return &UseCase{colors: colors}
}

type In struct {
	Account players.AccountID
	Color   players.Color
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	err := u.colors.SaveColor(ctx, in.Account, in.Color)
	if errors.Is(err, players.ErrNoProfile) {
		return err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return fmt.Errorf("failed to save the color: %w", err)
	}
	return nil
}
