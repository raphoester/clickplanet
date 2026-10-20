package antibot_place_shield

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error)
}

type Bans interface {
	EnforcedSentence(ctx context.Context, scope, account string) (antibot.Sentence, bool)
}

func New(implementation UseCase, bans Bans) *Decorator {
	return &Decorator{implementation: implementation, bans: bans}
}

type Decorator struct {
	implementation UseCase
	bans           Bans
}

func (d *Decorator) Execute(ctx context.Context, in place_shield_usecase.In) (bonuses.Held, error) {
	payer := clicks.PayerOf(ctx)
	if sentence, banned := d.bans.EnforcedSentence(ctx, payer.Scope, payer.Account); banned {
		return bonuses.Held{}, clicks.NewBanRefusal(sentence.Until)
	}

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
