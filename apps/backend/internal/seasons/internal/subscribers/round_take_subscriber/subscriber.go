package round_take_subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, take rounds.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber counts each tile taken in the rounds of their own, for their numbers.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.TileTaken] = Subscriber{}

var (
	errNoAccount = errors.New("the take has no account")
	errNoCountry = errors.New("the take has no country")
	errNoTime    = errors.New("the take has no time")
)

func (s Subscriber) Handle(ctx context.Context, event *planetv1.TileTaken) error {
	account, err := uuid.Parse(event.GetAccountId())
	if err != nil || account == uuid.Nil {
		return fmt.Errorf("tile %d: %w: %q", event.GetTileId(), errNoAccount, event.GetAccountId())
	}
	if event.GetCountry() == "" {
		return fmt.Errorf("tile %d: %w", event.GetTileId(), errNoCountry)
	}
	if err := event.GetTakenAt().CheckValid(); err != nil {
		return fmt.Errorf("tile %d: %w: %w", event.GetTileId(), errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, rounds.Take{ //nolint:wrapcheck // the use case named it.
		Account:  rounds.AccountID(account),
		Country:  rounds.Country(event.GetCountry()),
		At:       event.GetTakenAt().AsTime(),
		Position: event.GetPosition(),
	})
}
