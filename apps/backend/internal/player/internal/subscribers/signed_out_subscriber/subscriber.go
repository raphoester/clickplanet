// Package signed_out_subscriber hears auth.v1.SignedOut and takes the account off the roster.
package signed_out_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*authv1.SignedOut] = Subscriber{}

// Handle forgets the account's visit. A device still signed in to it announces again within 30s.
func (s Subscriber) Handle(ctx context.Context, event *authv1.SignedOut) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
