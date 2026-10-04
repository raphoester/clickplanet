package account_deleted_subscriber

import (
	"context"
	"errors"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, account messages.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*authv1.AccountDeleted] = Subscriber{}

var errNoAccount = errors.New("the deleted account has no id")

func (s Subscriber) Handle(ctx context.Context, event *authv1.AccountDeleted) error {
	account := messages.AccountIDOf(event.GetAccountId())
	if account == messages.NoAccount {
		return errNoAccount
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
