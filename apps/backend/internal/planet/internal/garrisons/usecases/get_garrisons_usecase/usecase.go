package get_garrisons_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
)

type Garrisons interface {
	Garrisons() []garrisons.Garrison
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

func New(garrisons Garrisons, owners Owners) *UseCase {
	return &UseCase{garrisons: garrisons, owners: owners}
}

type UseCase struct {
	garrisons Garrisons
	owners    Owners
}

func (u *UseCase) Execute(context.Context) []garrisons.Garrison {
	all := u.garrisons.Garrisons()
	standing := make([]garrisons.Garrison, 0, len(all))
	for _, garrison := range all {
		if owner, _ := u.owners.Owner(garrison.Tile); garrison.Standing(owner) > 0 {
			standing = append(standing, garrison)
		}
	}

	return standing
}
