// Package tile_taken_subscriber hears planet.v1.TileTaken and counts the tile for its account's and its scope's flag.
package tile_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/record_allegiance_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in record_allegiance_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*planetv1.TileTaken] = Subscriber{}

var (
	errNoCountry = errors.New("the take has no country")
	errNoTime    = errors.New("the take has no time")
)

// Handle refuses an event with no country or no time: it is the publisher's bug, and counting it would be a guess.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.TileTaken) error {
	if event.GetCountry() == "" {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), errNoCountry)
	}
	if err := event.GetTakenAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_allegiance_usecase.In{ //nolint:wrapcheck // the use case named it.
		Account: event.GetAccountId(),
		Scope:   event.GetScope(),
		Country: event.GetCountry(),
		At:      event.GetTakenAt().AsTime(),
	})
}
