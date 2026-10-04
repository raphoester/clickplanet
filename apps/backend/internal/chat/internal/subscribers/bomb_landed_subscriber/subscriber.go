package bomb_landed_subscriber

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

var _ cpbootstrap.Handler[*planetv1.BombLanded] = Subscriber{}

var errNoTime = errors.New("the bomb has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.BombLanded) error {
	if err := event.GetLandedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	payload, err := announcements.BombOf(event.GetCountry(), event.GetGround(), event.GetTileId(), event.GetCleared()).Payload()
	if err != nil {
		return err //nolint:wrapcheck // Payload named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_usecase.In{ //nolint:wrapcheck // the use case named it.
		Kind:    announcements.KindBomb,
		At:      event.GetLandedAt().AsTime(),
		Payload: payload,
	})
}
