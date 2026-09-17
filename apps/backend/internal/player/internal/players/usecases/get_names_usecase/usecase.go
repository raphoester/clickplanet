// Package get_names_usecase reads the names of many accounts at once, for another module.
package get_names_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Names interface {
	Names(accounts []players.AccountID) map[players.AccountID]players.Name
}

type UseCase struct {
	names Names
}

func New(names Names) *UseCase {
	return &UseCase{names: names}
}

// Execute leaves out every account with no name.
func (u *UseCase) Execute(accounts []players.AccountID) map[players.AccountID]players.Name {
	return u.names.Names(accounts)
}
