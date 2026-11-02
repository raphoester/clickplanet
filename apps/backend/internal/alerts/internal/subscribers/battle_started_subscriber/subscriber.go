package battle_started_subscriber

import (
	"context"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, start pings.Start) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.BattleStarted] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.BattleStarted) error {
	kind := pings.BattleStart
	if event.GetEndsSeason() {
		kind = pings.FinaleStart
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, pings.Start{ //nolint:wrapcheck // the use case named it.
		Key: pings.StartKey(event.GetRoundId()), Kind: kind, Season: event.GetSeason(), Times: event.GetTimes(),
		StartsAt: event.GetStartsAt().AsTime(), EndsAt: event.GetEndsAt().AsTime(),
	})
}
