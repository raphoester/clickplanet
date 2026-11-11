package bonus_claimed_subscriber

import (
	"context"
	"errors"
	"fmt"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_stats_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*bonusesv1.BonusClaimed] = Subscriber{}

var errNoTime = errors.New("the claim has no time")

func (s Subscriber) Handle(ctx context.Context, event *bonusesv1.BonusClaimed) error {
	account, err := accounts.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("bonus claimed: %w", err)
	}

	if err := event.GetClaimedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_stats_usecase.In{Account: account, Change: stats.BonusAt(event.GetClaimedAt().AsTime())}) //nolint:wrapcheck // the use case named it.
}
