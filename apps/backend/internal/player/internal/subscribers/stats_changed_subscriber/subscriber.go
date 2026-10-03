package stats_changed_subscriber

import (
	"context"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (titles.IDs, error)
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*playerv1.StatsChanged] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *playerv1.StatsChanged) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err = s.useCase.Execute(ctx, account)
	return err //nolint:wrapcheck // the use case named it.
}
