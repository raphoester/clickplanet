package account_deleted_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, account subscriptions.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*authv1.AccountDeleted] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *authv1.AccountDeleted) error {
	account, err := subscriptions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
