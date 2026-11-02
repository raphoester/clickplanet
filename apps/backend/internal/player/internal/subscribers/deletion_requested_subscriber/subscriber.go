package deletion_requested_subscriber

import (
	"context"
	"time"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID, decision departures.Decision) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*authv1.AccountDeletionRequested] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *authv1.AccountDeletionRequested) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the event's subject: the sentinel already says what is wrong.
	}
	var requestedAt time.Time
	if event.GetRequestedAt().IsValid() {
		requestedAt = event.GetRequestedAt().AsTime()
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account, departures.Leaves(requestedAt)) //nolint:wrapcheck // the use case named it.
}
