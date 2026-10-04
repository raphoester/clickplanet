package stats_changed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func subscriber(stats *inmemory_player_store.Store, held *inmemory_title_store.Store, linked players.AccountID) stats_changed_subscriber.Subscriber {
	accounts := titles.NewFakeAccounts()
	accounts.Create(linked, players.AccountOf(true, now))
	book := titles.NewBook(held, titles.CatalogOf([]titles.Title{titles.FakeTitle{Key: "first", Tiles: 1}}))
	return stats_changed_subscriber.New(award_titles_usecase.New(stats, accounts, book, cptime.NewFixedClock(now)))
}

func TestChangedStatsAreCheckedForTitles(t *testing.T) {
	stats, held := inmemory_player_store.New(), inmemory_title_store.New()
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, stats.RecordTake(t.Context(), account, now))

	err = subscriber(stats, held, account).Handle(t.Context(), &playerv1.StatsChanged{AccountId: account.String()})

	require.NoError(t, err)
	granted, err := held.Held(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"first"}, granted)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := subscriber(inmemory_player_store.New(), inmemory_title_store.New(), players.AccountID{15: 1}).Handle(t.Context(), &playerv1.StatsChanged{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
