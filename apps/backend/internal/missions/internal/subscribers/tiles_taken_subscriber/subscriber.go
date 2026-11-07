package tiles_taken_subscriber

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
	Execute(ctx context.Context, facts ...record_fact_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

var errNoTime = errors.New("the takes have no time")

// HandleBatch records each account's takes of the batch in one call, in the order they were told: an enclose of a
// continent is one write, not one per tile.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var told []missions.AccountID
	takes := map[missions.AccountID][]record_fact_usecase.In{}
	for _, event := range events {
		account, err := missions.AccountIDOf(event.GetAccountId())
		if err != nil {
			return fmt.Errorf("the takes at %d: %w", event.GetPosition(), err)
		}
		if err := event.GetTakenAt().CheckValid(); err != nil {
			return fmt.Errorf("the takes at %d: %w: %w", event.GetPosition(), errNoTime, err)
		}
		if len(event.GetTiles()) == 0 {
			continue
		}
		if _, known := takes[account]; !known {
			told = append(told, account)
		}
		at := event.GetTakenAt().AsTime()
		for _, tile := range event.GetTiles() {
			takes[account] = append(takes[account], record_fact_usecase.In{Account: account, At: at, Fact: missions.Take{
				Flag: event.GetCountry(), Previous: tile.GetPrevious(), Ground: tile.GetGround(),
				Position: event.GetPosition(), Index: tile.GetIndex(),
			}})
		}
	}

	for _, account := range told {
		if err := s.record(ctx, takes[account]); err != nil {
			return err
		}
	}
	return nil
}

func (s Subscriber) record(ctx context.Context, takes []record_fact_usecase.In) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, takes...) //nolint:wrapcheck // the use case named it.
}
