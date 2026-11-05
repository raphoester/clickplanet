package stats_changed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func subscriber(counted *inmemory_stats_store.Store, held *inmemory_title_store.Store, linked accounts.AccountID) stats_changed_subscriber.Subscriber {
	asked := titles.NewFakeAccounts()
	asked.Create(linked, accounts.AccountOf(true, now))
	book := titles.NewBook(held, titles.CatalogOf([]titles.Standalone{titles.FakeTitle{Key: "first", Tiles: 1}}))
	careers := titles.NewCareers(asked, inmemory_fight_store.New())
	return stats_changed_subscriber.New(award_titles_usecase.New(counted, careers, book, cptime.NewFixedClock(now)))
}

func TestChangedStatsAreCheckedForTitles(t *testing.T) {
	counted, held := inmemory_stats_store.New(), inmemory_title_store.New()
	account, err := accounts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, counted.Record(t.Context(), account, stats.TakeAt(0, 0, now)))

	err = subscriber(counted, held, account).Handle(t.Context(), &progressionv1.StatsChanged{AccountId: account.String()})

	require.NoError(t, err)
	granted, err := held.Held(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"first"}, granted)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := subscriber(inmemory_stats_store.New(), inmemory_title_store.New(), accounts.AccountID{15: 1}).Handle(t.Context(), &progressionv1.StatsChanged{})

	assert.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
