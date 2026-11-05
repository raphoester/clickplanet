package player_forgotten_subscriber

import (
	"context"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers"
)

type UseCase interface {
	Execute(ctx context.Context, account notes.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*playerv1.PlayerForgotten] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.PlayerForgotten) error {
	account, err := notes.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the rule named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
