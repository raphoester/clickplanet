package fortified_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.Fortified] = Subscriber{}

var errNoTime = errors.New("the fortify has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.Fortified) error {
	if err := event.GetFortifiedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	fortify := announcements.FortifyOf(event.GetPosition(), event.GetCountry(), event.GetGround(), event.GetLandmassId(),
		event.GetTiles())
	if !fortify.Newsworthy() {
		return nil
	}

	announcement, err := fortify.Announcement(event.GetFortifiedAt().AsTime())
	if err != nil {
		return err //nolint:wrapcheck // Announcement named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announcement) //nolint:wrapcheck // the use case named it.
}
