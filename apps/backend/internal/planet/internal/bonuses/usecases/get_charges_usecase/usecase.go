// Package get_charges_usecase reads what a caller holds, for a client that has just loaded or changed account.
package get_charges_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Charges interface {
	Held(holder bonuses.Holder) bonuses.Held
}

func New(charges Charges) *UseCase {
	return &UseCase{charges: charges}
}

type UseCase struct {
	charges Charges
}

// Execute derives the holder the way a claim does, so a caller reads the charges its claims granted.
func (u *UseCase) Execute(ctx context.Context) bonuses.Held {
	return u.charges.Held(bonuses.HolderOf(clicks.PayerOf(ctx)))
}
