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

func (u *UseCase) Execute(ctx context.Context) bonuses.Held {
	return u.charges.Held(bonuses.HolderOf(clicks.PayerOf(ctx)))
}
