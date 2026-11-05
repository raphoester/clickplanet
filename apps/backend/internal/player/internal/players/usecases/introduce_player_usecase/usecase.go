package introduce_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Codes interface {
	Assign(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	players players.PlayerStore
	codes   Codes
}

func New(store players.PlayerStore, codes Codes) *UseCase {
	return &UseCase{players: store, codes: codes}
}

// The player an account is, about to be shown: a guest is given its code the first time.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Player, error) {
	player, err := players.PlayerOf(ctx, u.players, account)
	if errors.Is(err, players.ErrNoGuestCode) {
		if err := u.codes.Assign(ctx, account); err != nil {
			return players.Player{}, fmt.Errorf("failed to give the guest a code: %w", err)
		}
		player, err = players.PlayerOf(ctx, u.players, account)
	}
	if err != nil {
		return players.Player{}, err //nolint:wrapcheck // PlayerOf says what failed.
	}
	return player, nil
}
