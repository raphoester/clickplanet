package get_charges_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Charges interface {
	Held(ctx context.Context, holder charges.Holder) (charges.Held, error)
}

func New(charges Charges) *UseCase {
	return &UseCase{charges: charges}
}

type UseCase struct {
	charges Charges
}

func (u *UseCase) Execute(ctx context.Context) (charges.Held, error) {
	return u.charges.Held(ctx, charges.HolderOf(clicks.PayerOf(ctx)))
}
