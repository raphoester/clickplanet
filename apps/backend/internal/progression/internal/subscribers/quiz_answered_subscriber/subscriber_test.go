package quiz_answered_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/quiz_answered_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func subscriber() (quiz_answered_subscriber.Subscriber, *inmemory_stats_store.Store) {
	store := inmemory_stats_store.New()
	return quiz_answered_subscriber.New(record_stats_usecase.New(store, cpredis.NewRecordedEvents())), store
}

func statsOf(t *testing.T, store *inmemory_stats_store.Store) stats.StatsSnapshot {
	t.Helper()

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	return stats.TakeSnapshot()
}

func TestOnlyARightAnswerCounts(t *testing.T) {
	handler, store := subscriber()

	require.NoError(t, handler.Handle(t.Context(), &planetv1.QuizAnswered{
		AccountId: account, Correct: true, AnsweredAt: timestamppb.New(at),
	}))
	require.NoError(t, handler.Handle(t.Context(), &planetv1.QuizAnswered{
		AccountId: account, Correct: false, AnsweredAt: timestamppb.New(at.Add(time.Minute)),
	}))

	assert.Equal(t, uint64(1), statsOf(t, store).RightAnswers)
}

func TestAnAnswerWithNoAccountIsRefused(t *testing.T) {
	handler, _ := subscriber()

	assert.ErrorIs(t, handler.Handle(t.Context(), &planetv1.QuizAnswered{Correct: true}), accounts.ErrInvalidAccount)
}
