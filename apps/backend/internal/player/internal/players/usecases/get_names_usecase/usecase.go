// Package get_names_usecase reads the names of many accounts at once, for another module.
package get_names_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Names interface {
	Names(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error)
}

type UseCase struct {
	names Names
}

func New(names Names) *UseCase {
	return &UseCase{names: names}
}

// Execute leaves out every account with no name.
func (u *UseCase) Execute(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error) {
	names, err := u.names.Names(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the names: %w", err)
	}
	return names, nil
}
