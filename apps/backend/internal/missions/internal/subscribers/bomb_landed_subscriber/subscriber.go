package bomb_landed_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_fact_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.BombLanded] = Subscriber{}

var errNoTime = errors.New("the bomb has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.BombLanded) error {
	if event.GetAccountId() == "" {
		return nil
	}
	account, err := missions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("bomb at tile %d: %w", event.GetTileId(), err)
	}
	if err := event.GetLandedAt().CheckValid(); err != nil {
		return fmt.Errorf("bomb at tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_fact_usecase.In{ //nolint:wrapcheck // the use case named it.
		Account: account,
		At:      event.GetLandedAt().AsTime(),
		Fact:    missions.BombLanded{Ground: event.GetGround(), Position: event.GetPosition()},
	})
}
