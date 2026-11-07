package tiles_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, takes []standings.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

var errNoTime = errors.New("the takes have no time")

// HandleBatch counts the takes of the batch in one call, in the order they were told: one write per book and account,
// not one per tile.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var takes []standings.Take
	for _, event := range events {
		account, err := standings.AccountIDOf(event.GetAccountId())
		if err != nil {
			return fmt.Errorf("the takes at %d: %w", event.GetPosition(), err)
		}
		if event.GetCountry() == "" {
			return fmt.Errorf("the takes at %d: %w", event.GetPosition(), standings.ErrNoCountry)
		}
		if err := event.GetTakenAt().CheckValid(); err != nil {
			return fmt.Errorf("the takes at %d: %w: %w", event.GetPosition(), errNoTime, err)
		}
		at := event.GetTakenAt().AsTime()
		for _, tile := range event.GetTiles() {
			takes = append(takes, standings.Take{
				Account:  account,
				Country:  standings.Country(event.GetCountry()),
				At:       at,
				Position: event.GetPosition(),
				Index:    tile.GetIndex(),
			})
		}
	}
	if len(takes) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, takes) //nolint:wrapcheck // the use case named it.
}
