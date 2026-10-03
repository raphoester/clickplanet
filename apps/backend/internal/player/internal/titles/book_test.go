package titles_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
)

var (
	at  = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func holding(t *testing.T, ids ...titles.ID) *inmemory_title_store.Store {
	t.Helper()

	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: ids}, at))
	return store
}

func TestUnheldIsWhatTheCareerEarnsAndTheAccountDoesNotHold(t *testing.T) {
	unheld, err := titles.NewBook(holding(t, "first"), catalog).Unheld(t.Context(), ada, tiles(3))

	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"third"}, unheld)
}

func TestAGuestEarnsNothingAndNoStoreIsRead(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))

	unheld, err := titles.NewBook(store, catalog).Unheld(t.Context(), ada, titles.Career{Stats: players.Stats{TilesTaken: 3}})

	require.NoError(t, err)
	assert.Empty(t, unheld)
}

func TestGrantedTitlesAreHeld(t *testing.T) {
	store := inmemory_title_store.New()

	require.NoError(t, titles.NewBook(store, catalog).Grant(t.Context(), ada, titles.IDs{"first"}, at))

	held, err := store.Held(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"first"}, held)
}

func TestShownIsWhatTheCatalogShowsOfTheTitlesHeld(t *testing.T) {
	shown, err := titles.NewBook(holding(t, "badge", "low", "mid"), tracks).Shown(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, []titles.Standing{{Title: badge}, {Title: mid, Place: place(ladder, 2)}}, shown)
}

func TestProgressIsEachTrackMeasuredOnTheCareerAgainstTheTitlesHeld(t *testing.T) {
	progress, err := titles.NewBook(holding(t, "badge", "low"), tracks).Progress(t.Context(), ada, tiles(3))

	require.NoError(t, err)
	assert.Equal(t, tracks.Progress(tiles(3), titles.IDs{"badge", "low"}), progress)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))
	book := titles.NewBook(store, tracks)

	_, err := book.Shown(t.Context(), ada)
	require.Error(t, err)
	_, err = book.Progress(t.Context(), ada, tiles(3))
	require.Error(t, err)
	_, err = book.Unheld(t.Context(), ada, tiles(3))
	assert.Error(t, err)
}
