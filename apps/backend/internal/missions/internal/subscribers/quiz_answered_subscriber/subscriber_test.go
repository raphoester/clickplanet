package quiz_answered_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/record_fact_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers/quiz_answered_subscriber"
)

type recordingUseCase struct {
	facts []record_fact_usecase.In
}

func (u *recordingUseCase) Execute(_ context.Context, in record_fact_usecase.In) error {
	u.facts = append(u.facts, in)
	return nil
}

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var (
	at    = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	id, _ = missions.AccountIDOf(account)
)

func TestARightAnswerReachesTheMissionsAndAWrongOneDoesNot(t *testing.T) {
	useCase := &recordingUseCase{}
	subscriber := quiz_answered_subscriber.New(useCase)

	require.NoError(t, subscriber.Handle(t.Context(), &planetv1.QuizAnswered{AccountId: account, Correct: true, AnsweredAt: timestamppb.New(at)}))
	require.NoError(t, subscriber.Handle(t.Context(), &planetv1.QuizAnswered{AccountId: account, AnsweredAt: timestamppb.New(at)}))

	assert.Equal(t, []record_fact_usecase.In{{Account: id, At: at, Fact: missions.RightAnswer{At: at}}}, useCase.facts)
}

func TestAnAnswerWithNoAccountOrNoTimeIsRefused(t *testing.T) {
	subscriber := quiz_answered_subscriber.New(&recordingUseCase{})

	require.ErrorIs(t, subscriber.Handle(t.Context(), &planetv1.QuizAnswered{Correct: true, AnsweredAt: timestamppb.New(at)}),
		missions.ErrInvalidAccount)
	assert.Error(t, subscriber.Handle(t.Context(), &planetv1.QuizAnswered{AccountId: account, Correct: true}))
}
