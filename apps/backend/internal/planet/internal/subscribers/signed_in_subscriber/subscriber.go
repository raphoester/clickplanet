package signed_in_subscriber

import (
	"context"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, holder bonuses.Holder) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*authv1.SignedIn] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *authv1.SignedIn) error {
	if !event.GetGuestLinked() {
		return nil
	}
	holder, err := bonuses.ParseHolder(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is what the refill is for: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, holder) //nolint:wrapcheck // the use case named it.
}
