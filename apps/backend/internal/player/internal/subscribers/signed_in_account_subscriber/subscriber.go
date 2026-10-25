package signed_in_account_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase[R any] interface {
	Execute(ctx context.Context, account players.AccountID) (R, error)
}

func New[R any](useCase UseCase[R]) Subscriber[R] {
	return Subscriber[R]{useCase: useCase}
}

type Subscriber[R any] struct {
	useCase UseCase[R]
}

var _ cpbootstrap.Handler[*authv1.SignedIn] = Subscriber[players.Profile]{}

func (s Subscriber[R]) Handle(ctx context.Context, event *authv1.SignedIn) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err = s.useCase.Execute(ctx, account)
	return err //nolint:wrapcheck // the use case named it.
}
