package titles_test

import (
	"context"
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

func TestAwardGrantsWhatTheStatsEarn(t *testing.T) {
	store := inmemory_title_store.New()

	require.NoError(t, titles.NewBook(store, catalog).Award(t.Context(), ada, tiles(3), at))

	held, err := store.Held(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, titles.IDs{"first", "third"}, held)
}

type refusingGrants struct {
	*inmemory_title_store.Store
}

func (refusingGrants) Grant(context.Context, titles.Holdings, time.Time) error {
	return errors.New("nothing was to be granted")
}

func TestAwardWritesNothingWhenEveryEarnedTitleIsHeld(t *testing.T) {
	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: {"first"}}, at))

	err := titles.NewBook(refusingGrants{store}, catalog).Award(t.Context(), ada, tiles(2), at)

	assert.NoError(t, err)
}

func TestTheTitlesOfAnAccountAreTheCatalogsObjects(t *testing.T) {
	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{ada: {"third", "first"}}, at))

	held, err := titles.NewBook(store, catalog).TitlesOf(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, []titles.Title{first, third}, held)
}

func TestAGuestsAwardTouchesNoStore(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))
	guest := titles.Career{Stats: players.Stats{TilesTaken: 3}}

	assert.NoError(t, titles.NewBook(store, catalog).Award(t.Context(), ada, guest, at))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))
	book := titles.NewBook(store, catalog)

	_, err := book.TitlesOf(t.Context(), ada)
	require.Error(t, err)
	assert.Error(t, book.Award(t.Context(), ada, tiles(3), at))
}
