package round_closed_subscriber

import (
	"context"
	"errors"
	"fmt"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*seasonsv1.RoundClosed] = Subscriber{}

var errNoTime = errors.New("the round has no end")

func (s Subscriber) Handle(ctx context.Context, event *seasonsv1.RoundClosed) error {
	if err := event.GetEndedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	places := make([]announcements.Place, 0, len(event.GetResults()))
	for _, result := range event.GetResults() {
		places = append(places, announcements.PlaceOf(result.GetCountry(), result.GetRank(), result.GetPoints()))
	}
	payload, err := announcements.RoundOf(event.GetNumber(), event.GetFinale(), places).Payload()
	if err != nil {
		return err //nolint:wrapcheck // Payload named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_usecase.In{ //nolint:wrapcheck // the use case named it.
		Kind:    announcements.KindRound,
		At:      event.GetEndedAt().AsTime(),
		Payload: payload,
	})
}
