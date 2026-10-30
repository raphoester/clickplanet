package player_created_subscriber

import (
	"context"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
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

var _ cpredis.Handler[*playerv1.PlayerCreated] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.PlayerCreated) error {
	holder, err := bonuses.ParseHolder(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, holder) //nolint:wrapcheck // the use case named it.
}
