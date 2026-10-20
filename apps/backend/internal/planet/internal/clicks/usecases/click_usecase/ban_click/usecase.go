package ban_click

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Bans interface {
	EnforcedSentence(ctx context.Context, scope, account string) (antibot.Sentence, bool)
}

func New(implementation click_usecase.IUseCase, bans Bans, registerer prometheus.Registerer) *UseCase {
	refused := promauto.With(registerer).NewCounter(prometheus.CounterOpts{
		Name: "banned_clicks",
		Help: "Clicks refused because their caller is banned",
	})

	return &UseCase{implementation: implementation, bans: bans, refused: refused}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	bans           Bans
	refused        prometheus.Counter
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	payer := clicks.PayerOf(ctx)
	if sentence, banned := u.bans.EnforcedSentence(ctx, payer.Scope, payer.Account); banned {
		u.refused.Inc()
		return click_usecase.Out{}, clicks.NewBanRefusal(sentence.Until)
	}

	return u.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
