package battle_ended_subscriber

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

var _ cpredis.Handler[*gamemasterv1.BattleEnded] = Subscriber{}

var errNoTime = errors.New("the battle has no end")

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.BattleEnded) error {
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
		Round:   announcements.BattleOf(event.GetSeason(), event.GetNumber(), event.GetEndsSeason(), event.GetTimes()),
		RoundID: round,
		At:      event.GetEndedAt().AsTime(),
	})
}
