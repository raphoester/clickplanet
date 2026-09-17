// Package account_deleted_subscriber hears auth.v1.AccountDeleted and forgets the account's profile and stats.
package account_deleted_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(account players.AccountID)
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*authv1.AccountDeleted] = Subscriber{}

func (s Subscriber) Handle(_ context.Context, event *authv1.AccountDeleted) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	s.useCase.Execute(account)
	return nil
}
