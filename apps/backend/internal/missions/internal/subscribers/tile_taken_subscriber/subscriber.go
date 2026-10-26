package tile_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, take missions.Take) error
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
	account, err := missions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), err)
	}
	if err := event.GetTakenAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, missions.Take{ //nolint:wrapcheck // the use case named it.
		Account:  account,
		Flag:     event.GetCountry(),
		Previous: event.GetPrevious(),
		Ground:   event.GetGround(),
		At:       event.GetTakenAt().AsTime(),
		Position: event.GetPosition(),
	})
}
