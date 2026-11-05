package get_charges_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Charges interface {
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

func New(charges Charges) *UseCase {
	return &UseCase{charges: charges}
}

type UseCase struct {
	charges Charges
}

func (u *UseCase) Execute(ctx context.Context) (bonuses.Held, error) {
	return u.charges.Held(ctx, bonuses.HolderOf(clicks.PayerOf(ctx)))
}
