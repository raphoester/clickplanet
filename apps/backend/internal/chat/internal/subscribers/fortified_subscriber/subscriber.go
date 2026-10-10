package fortified_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
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

var _ cpbootstrap.Handler[*planetv1.Fortified] = Subscriber{}

var errNoTime = errors.New("the fortify has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.Fortified) error {
	if err := event.GetFortifiedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	fortify := announcements.FortifyOf(event.GetCountry(), event.GetGround(), event.GetLandmassId(), event.GetTiles())
	if !fortify.Newsworthy() {
		return nil
	}

	payload, err := fortify.Payload()
	if err != nil {
		return err //nolint:wrapcheck // Payload named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_usecase.In{ //nolint:wrapcheck // the use case named it.
		Kind:    announcements.KindFortify,
		At:      event.GetFortifiedAt().AsTime(),
		Payload: payload,
	})
}
