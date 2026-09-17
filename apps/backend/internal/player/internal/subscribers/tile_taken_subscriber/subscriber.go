// Package tile_taken_subscriber hears planet.v1.TileTaken and counts the tile on the account's stats.
package tile_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(in record_take_usecase.In)
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*planetv1.TileTaken] = Subscriber{}

var errNoTime = errors.New("the take has no time")

// Handle refuses an event with no account or no time: it is planet's bug, and counting it would be a guess.
func (s Subscriber) Handle(_ context.Context, event *planetv1.TileTaken) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), err)
	}
	if err := event.GetTakenAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	s.useCase.Execute(record_take_usecase.In{Account: account, At: event.GetTakenAt().AsTime()})
	return nil
}
