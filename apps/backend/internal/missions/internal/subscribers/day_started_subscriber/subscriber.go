package day_started_subscriber

import (
	"context"
	"fmt"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, day days.Day) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.DayStarted] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.DayStarted) error {
	day, err := days.DayOf(days.DaySnapshot{
		ID: days.DayID(event.GetRoundId()), StartsAt: event.GetStartsAt().AsTime(), EndsAt: event.GetEndsAt().AsTime(),
	})
	if err != nil {
		return fmt.Errorf("day %d of season %d: %w", event.GetDay(), event.GetSeason(), err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, day) //nolint:wrapcheck // the use case named it.
}
