package tile_taken_fronts_subscriber

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, take fronts.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*planetv1.TileTaken] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *planetv1.TileTaken) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), err)
	}
	take, err := fronts.NewTake(account, fronts.Country(event.GetCountry()), fronts.Country(event.GetPreviousCountry()))
	if err != nil {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, take) //nolint:wrapcheck // the use case named it.
}
