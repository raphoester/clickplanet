package season_ended_subscriber

import (
	"context"
	"errors"
	"fmt"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_champion_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_champion_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.SeasonEnded] = Subscriber{}

var errNoTime = errors.New("the season has no end")

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.SeasonEnded) error {
	if err := event.GetEndedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	book, err := announcements.BookIDOf(event.GetBookId())
	if err != nil {
		return err //nolint:wrapcheck // BookIDOf named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_champion_usecase.In{ //nolint:wrapcheck // the use case named it.
		Season: event.GetSeason(),
		Book:   book,
		At:     event.GetEndedAt().AsTime(),
	})
}
