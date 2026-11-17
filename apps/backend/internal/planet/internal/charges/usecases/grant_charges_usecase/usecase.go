package grant_charges_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

type Charger interface {
	Grant(ctx context.Context, holder charges.Holder, kind charges.Kind, amount int) (charges.Gain, error)
	Held(ctx context.Context, holder charges.Holder) (charges.Held, error)
}

type Streams interface {
	PublishGranted(holder charges.Holder, gain charges.Gain)
}

type In struct {
	Account string
	Grant   charges.Held
}

type Out struct {
	Holder charges.Holder
	Before charges.Held
	After  charges.Held
}

func New(charger Charger, streams Streams) *UseCase {
	return &UseCase{charger: charger, streams: streams}
}

type UseCase struct {
	charger Charger
	streams Streams
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	holder, err := charges.ParseHolder(in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}
	if err := in.Grant.GrantError(); err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}

	before, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}
	var paid []charges.Kind
	for _, kind := range charges.Kinds {
		if amount := in.Grant.Count(kind); amount > 0 {
			gain, err := u.charger.Grant(ctx, holder, kind, amount)
			if err != nil {
				return Out{}, err
			}
			paid = append(paid, gain.Paid...)
		}
	}
	after, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}

	u.streams.PublishGranted(holder, charges.Gain{Paid: paid, Before: before, After: after})
	return Out{Holder: holder, Before: before, After: after}, nil
}
