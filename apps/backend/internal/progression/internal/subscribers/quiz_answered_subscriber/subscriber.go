package quiz_answered_subscriber

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

var _ cpredis.Handler[*bonusesv1.QuizAnswered] = Subscriber{}

var errNoTime = errors.New("the answer has no time")

func (s Subscriber) Handle(ctx context.Context, event *bonusesv1.QuizAnswered) error {
	account, err := accounts.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("quiz answered: %w", err)
	}
	if !event.GetCorrect() {
		return nil
	}

	if err := event.GetAnsweredAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_stats_usecase.In{Account: account, Change: stats.RightAnswerAt(event.GetAnsweredAt().AsTime())}) //nolint:wrapcheck // the use case named it.
}
