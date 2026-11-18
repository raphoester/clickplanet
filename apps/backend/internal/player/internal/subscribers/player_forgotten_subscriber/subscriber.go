package player_forgotten_subscriber

import (
	"context"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Player's own PlayerForgotten, for what it keeps of an account outside the forget's transaction.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*playerv1.PlayerForgotten] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.PlayerForgotten) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the event's subject: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
