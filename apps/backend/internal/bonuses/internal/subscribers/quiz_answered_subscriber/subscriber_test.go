package quiz_answered_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/quiz_answered_subscriber"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type paid struct {
	answers []bonuses.Answer
}

func (p *paid) Execute(_ context.Context, answer bonuses.Answer) error {
	p.answers = append(p.answers, answer)
	return nil
}

func TestARightAnswerIsPaidUnderItsToken(t *testing.T) {
	useCase := &paid{}

	require.NoError(t, quiz_answered_subscriber.New(useCase).Handle(t.Context(), &quizzesv1.QuizAnswered{
		AccountId: ada, Correct: true, Token: "a-token", Welcome: true,
	}))

	assert.Equal(t, []bonuses.Answer{{Account: ada, Token: "a-token", Welcome: true}}, useCase.answers)
}

func TestAWrongAnswerIsPaidNothing(t *testing.T) {
	useCase := &paid{}

	require.NoError(t, quiz_answered_subscriber.New(useCase).Handle(t.Context(), &quizzesv1.QuizAnswered{
		AccountId: ada, Token: "a-token",
	}))

	assert.Empty(t, useCase.answers)
}

func TestARightAnswerWithNoAccountOrNoTokenIsRefused(t *testing.T) {
	useCase := &paid{}
	subscriber := quiz_answered_subscriber.New(useCase)

	require.ErrorIs(t, subscriber.Handle(t.Context(), &quizzesv1.QuizAnswered{AccountId: "nope", Correct: true, Token: "t"}),
		bonuses.ErrInvalidAccount)
	require.Error(t, subscriber.Handle(t.Context(), &quizzesv1.QuizAnswered{AccountId: ada, Correct: true}))
	assert.Empty(t, useCase.answers)
}
