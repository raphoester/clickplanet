package quiz_answered_subscriber

import (
	"context"
	"errors"
	"fmt"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, answer bonuses.Answer) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber pays each right answer the quizzes tell; a wrong one pays nothing.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*quizzesv1.QuizAnswered] = Subscriber{}

var errNoToken = errors.New("the answer names no quiz")

func (s Subscriber) Handle(ctx context.Context, event *quizzesv1.QuizAnswered) error {
	if !event.GetCorrect() {
		return nil
	}
	account, err := bonuses.ParseAccount(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("quiz answered: %w", err)
	}
	if event.GetToken() == "" {
		return errNoToken
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, bonuses.Answer{ //nolint:wrapcheck // the use case named it.
		Account: account, Token: event.GetToken(), Welcome: event.GetWelcome(),
	})
}
