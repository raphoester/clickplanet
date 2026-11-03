package day_ended_subscriber

import (
	"context"
	"errors"
	"fmt"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, day pings.EndedDay) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.DayEnded] = Subscriber{}

var (
	errNoEnd   = errors.New("the day has no end")
	errNoRound = errors.New("the day names no round")
)

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.DayEnded) error {
	if err := event.GetEndedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoEnd, err)
	}
	if event.GetRoundId() == "" {
		return errNoRound
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, pings.EndedDay{ //nolint:wrapcheck // the use case named it.
		Round: pings.RoundID(event.GetRoundId()), Season: event.GetSeason(), Number: event.GetDay(),
		EndedAt: event.GetEndedAt().AsTime(),
	})
}
