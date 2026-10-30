package get_ad_refill_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type AdRefills interface {
	AvailableAt(ctx context.Context, payer clicks.Payer) (time.Time, error)
}

func New(adRefills AdRefills) *UseCase {
	return &UseCase{adRefills: adRefills}
}

type UseCase struct {
	adRefills AdRefills
}

func (u *UseCase) Execute(ctx context.Context) (time.Time, error) {
	return u.adRefills.AvailableAt(ctx, clicks.PayerOf(ctx)) //nolint:wrapcheck // the ad refills named it.
}
