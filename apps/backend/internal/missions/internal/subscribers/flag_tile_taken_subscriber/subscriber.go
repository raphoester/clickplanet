package flag_tile_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, take flagmissions.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.TileTaken] = Subscriber{}

var errNoTime = errors.New("the take has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.TileTaken) error {
	if err := event.GetTakenAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, flagmissions.Take{ //nolint:wrapcheck // the use case named it.
		Flag: event.GetCountry(), Ground: event.GetGround(), Position: event.GetPosition(), Index: event.GetIndex(),
		At: event.GetTakenAt().AsTime(),
	})
}
