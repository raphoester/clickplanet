package day_ended_subscriber

import (
	"context"
	"errors"
	"fmt"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_round_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.DayEnded] = Subscriber{}

var errNoTime = errors.New("the day has no end")

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.DayEnded) error {
	if err := event.GetEndedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	round, err := announcements.RoundIDOf(event.GetRoundId())
	if err != nil {
		return err //nolint:wrapcheck // RoundIDOf named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_round_usecase.In{ //nolint:wrapcheck // the use case named it.
		Round:   announcements.DayOf(event.GetSeason(), event.GetDay()),
		RoundID: round,
		At:      event.GetEndedAt().AsTime(),
	})
}
