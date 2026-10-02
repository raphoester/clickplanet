package stats_changed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/stats_changed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

func subscriber(store *inmemory_player_store.Store) stats_changed_subscriber.Subscriber {
	book := players.NewTitleBook(store, players.Catalog{players.FakeTitle{Key: "first", Tiles: 1}})
	return stats_changed_subscriber.New(award_titles_usecase.New(store, players.NewFakeAccounts(), book, cptime.NewFixedClock(now)))
}

func TestChangedStatsAreCheckedForTitles(t *testing.T) {
	store := inmemory_player_store.New()
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.RecordTake(t.Context(), account, now))

	err = subscriber(store).Handle(t.Context(), &playerv1.StatsChanged{AccountId: account.String()})

	require.NoError(t, err)
	titles, err := store.Titles(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, players.TitleIDs{"first"}, titles)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	store := inmemory_player_store.New()

	err := subscriber(store).Handle(t.Context(), &playerv1.StatsChanged{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
