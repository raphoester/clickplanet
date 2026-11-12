package player_named_subscriber

import (
	"context"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// A call of the judge, its retry included: a name is a few seconds of Claude Haiku.
const Timeout = time.Minute

type UseCase interface {
	Execute(ctx context.Context, account casts.AccountID) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*playerv1.PlayerNamed] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.PlayerNamed) error {
	account, err := casts.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, account) //nolint:wrapcheck // the use case named it.
}
