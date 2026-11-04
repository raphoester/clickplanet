package record_stats_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var (
	at  = time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC)
	ada = accounts.AccountID{15: 1}
)

func TestEachChangeIsRecordedAndSaysTheStatsChanged(t *testing.T) {
	store := inmemory_stats_store.New()
	events := cpredis.NewRecordedEvents()
	useCase := record_stats_usecase.New(store, events)

	require.NoError(t, useCase.Execute(t.Context(), record_stats_usecase.In{Account: ada, Change: stats.TakeAt(0, 0, at)}))
	require.NoError(t, useCase.Execute(t.Context(), record_stats_usecase.In{Account: ada, Change: stats.BombAt(1)}))

	counted, err := store.Stats(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, stats.NewStats(ada).WithTake(0, 0, at).WithBomb(1), counted)
	require.Len(t, events.Published(), 2)
	assert.True(t, proto.Equal(&progressionv1.StatsChanged{AccountId: ada.String()}, events.Published()[1]))
}

func TestAStoreFailureIsAnErrorAndSaysNothing(t *testing.T) {
	store := inmemory_stats_store.New()
	store.FailWith(errors.New("postgres is down"))
	events := cpredis.NewRecordedEvents()

	err := record_stats_usecase.New(store, events).Execute(t.Context(), record_stats_usecase.In{Account: ada, Change: stats.MessageAt(at)})

	require.Error(t, err)
	assert.Empty(t, events.Published())
}
