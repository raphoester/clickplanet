package bonus_claimed_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_fact_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.BonusClaimed] = Subscriber{}

var errNoTime = errors.New("the claim has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.BonusClaimed) error {
	account, err := missions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("box caught: %w", err)
	}
	if err := event.GetClaimedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	at := event.GetClaimedAt().AsTime()
	return s.useCase.Execute(ctx, record_fact_usecase.In{Account: account, At: at, Fact: missions.BoxCaught{At: at}}) //nolint:wrapcheck // the use case named it.
}
