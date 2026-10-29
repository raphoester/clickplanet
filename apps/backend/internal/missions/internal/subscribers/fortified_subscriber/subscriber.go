package fortified_subscriber

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

var _ cpredis.Handler[*planetv1.Fortified] = Subscriber{}

var errNoTime = errors.New("the fortify has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.Fortified) error {
	if event.GetAccountId() == "" {
		return nil
	}
	account, err := missions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("fortify of landmass %d: %w", event.GetLandmassId(), err)
	}
	if err := event.GetFortifiedAt().CheckValid(); err != nil {
		return fmt.Errorf("fortify of landmass %d: %w: %w", event.GetLandmassId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_fact_usecase.In{ //nolint:wrapcheck // the use case named it.
		Account: account,
		At:      event.GetFortifiedAt().AsTime(),
		Fact:    missions.Fortified{Tiles: event.GetTiles(), Position: event.GetPosition()},
	})
}
