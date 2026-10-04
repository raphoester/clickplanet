package gift_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type Tempo interface {
	Rules() tempo.Rules
}

type Gifts interface {
	Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error
}

type Bank interface {
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
}

type Charger interface {
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
}

func New(
	implementation click_usecase.IUseCase,
	tempo Tempo,
	gifts Gifts,
	bank Bank,
	charger Charger,
	buckets clicks.Buckets,
) *UseCase {
	return &UseCase{implementation: implementation, tempo: tempo, gifts: gifts, bank: bank, charger: charger, buckets: buckets}
}

// Outside the shadow ban: a gift a banned caller never got would tell it it is banned.
type UseCase struct {
	implementation click_usecase.IUseCase
	tempo          Tempo
	gifts          Gifts
	bank           Bank
	charger        Charger
	buckets        clicks.Buckets
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil {
		return out, err //nolint:wrapcheck // a decorator passes it on.
	}

	gift, gifting := u.tempo.Rules().Gift()
	payer := clicks.PayerOf(ctx)
	if !gifting || !gift.OwedTo(payer) {
		return out, nil
	}

	holder := bonuses.HolderOf(payer)
	if u.gifts.Give(ctx, gift.Tag(), holder) != nil {
		return out, nil
	}

	for _, key := range u.buckets.Bank(payer) {
		_, _ = u.bank.Fill(key)
	}
	u.charger.Grant(holder, bonuses.KindBomb, 1)
	out.Gift = true

	return out, nil
}
