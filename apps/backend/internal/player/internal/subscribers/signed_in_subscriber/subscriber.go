package signed_in_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, from, to players.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*authv1.SignedIn] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *authv1.SignedIn) error {
	if event.GetPreviousAccountId() == "" {
		return nil
	}

	from, err := players.AccountIDOf(event.GetPreviousAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel already says what is wrong.
	}
	to, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, from, to) //nolint:wrapcheck // the use case named it.
}
