package antibot_drop_bomb

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
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

func (d *Decorator) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	payer := clicks.PayerOf(ctx)
	if sentence, banned := d.bans.EnforcedSentence(ctx, payer.Scope, payer.Account); banned {
		return clicks.Blast{}, clicks.NewBanRefusal(sentence.Until)
	}

	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
