package round_take_subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, takes []books.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber counts each tile taken in the rounds, for their numbers.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

var (
	errNoAccount = errors.New("the takes have no account")
	errNoCountry = errors.New("the takes have no country")
	errNoTime    = errors.New("the takes have no time")
)

// HandleBatch counts the takes of the batch in one call, in the order they were told: one write, not one per tile.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var takes []books.Take
	for _, event := range events {
		account, err := uuid.Parse(event.GetAccountId())
		if err != nil || account == uuid.Nil {
			return fmt.Errorf("the takes at %d: %w: %q", event.GetPosition(), errNoAccount, event.GetAccountId())
		}
		if event.GetCountry() == "" {
			return fmt.Errorf("the takes at %d: %w", event.GetPosition(), errNoCountry)
		}
		if err := event.GetTakenAt().CheckValid(); err != nil {
			return fmt.Errorf("the takes at %d: %w: %w", event.GetPosition(), errNoTime, err)
		}
		at := event.GetTakenAt().AsTime()
		for _, tile := range event.GetTiles() {
			takes = append(takes, books.Take{
				Account:  books.AccountID(account),
				Country:  books.Country(event.GetCountry()),
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
