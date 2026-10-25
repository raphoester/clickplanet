package bomb_landed_subscriber

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in record_stats_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*planetv1.BombLanded] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *planetv1.BombLanded) error {
	if event.GetAccountId() == "" {
		return nil
	}
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("bomb at tile %d: %w", event.GetTileId(), err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_stats_usecase.In{Account: account, Change: players.BombAt(event.GetPosition())}) //nolint:wrapcheck // the use case named it.
}
