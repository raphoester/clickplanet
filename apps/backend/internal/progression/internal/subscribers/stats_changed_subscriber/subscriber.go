package stats_changed_subscriber

import (
	"context"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// UseCase is what the stats now give the account: titles, a level. What it gave is not the subscriber's.
type UseCase[Given any] interface {
	Execute(ctx context.Context, account accounts.AccountID) (Given, error)
}

func New[Given any](useCase UseCase[Given]) Subscriber[Given] {
	return Subscriber[Given]{useCase: useCase}
}

type Subscriber[Given any] struct {
	useCase UseCase[Given]
}

var _ cpredis.Handler[*progressionv1.StatsChanged] = Subscriber[struct{}]{}

func (s Subscriber[Given]) Handle(ctx context.Context, event *progressionv1.StatsChanged) error {
	account, err := accounts.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the id is the whole event: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err = s.useCase.Execute(ctx, account)
	return err //nolint:wrapcheck // the use case named it.
}
