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

func TestTheShowcaseIsTheWornTitleAndWhatIsShown(t *testing.T) {
	showcase, err := titles.NewBook(holding(t, "badge", "low", "mid"), tracks).Showcase(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, titles.Showcase{
		Worn:  titles.Standing{Title: mid, Place: place(ladder, 2)},
		Shown: []titles.Standing{{Title: badge}, {Title: mid, Place: place(ladder, 2)}},
	}, showcase)
}

func TestWearingAShownTitleChangesTheShowcase(t *testing.T) {
	book := titles.NewBook(holding(t, "badge", "low", "mid"), tracks)

	require.NoError(t, book.Wear(t.Context(), ada, "badge", at))

	showcase, err := book.Showcase(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, titles.Standing{Title: badge}, showcase.Worn)
}

func TestATitleNotShownCannotBeWorn(t *testing.T) {
	store := holding(t, "badge", "low", "mid")
	book := titles.NewBook(store, tracks)

	for _, id := range []titles.ID{"low", "high", "retired"} {
		assert.ErrorIs(t, book.Wear(t.Context(), ada, id, at), titles.ErrNotWearable, id)
	}
	worn, err := store.Worn(t.Context(), ada)
	require.NoError(t, err)
	assert.Empty(t, worn, "a refused title writes nothing")
}

func TestTheDashboardIsTheWornTitleWhatCanBeWornAndEachTracksProgress(t *testing.T) {
	dashboard, err := titles.NewBook(holding(t, "badge", "low"), tracks).Dashboard(t.Context(), ada, tiles(3))

	require.NoError(t, err)
	assert.Equal(t, titles.Standing{Title: low, Place: place(ladder, 1)}, dashboard.Worn)
	assert.Equal(t, []titles.Standing{{Title: badge}, {Title: low, Place: place(ladder, 1)}}, dashboard.Wearable)
	assert.Equal(t, tracks.Progress(tiles(3), titles.IDs{"badge", "low"}), dashboard.Tracks)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))
	book := titles.NewBook(store, tracks)

	_, err := book.Showcase(t.Context(), ada)
	require.Error(t, err)
	_, err = book.Dashboard(t.Context(), ada, tiles(3))
	require.Error(t, err)
	_, err = book.Unheld(t.Context(), ada, tiles(3))
	require.Error(t, err)
	assert.Error(t, book.Wear(t.Context(), ada, "badge", at))
}
