package players_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
)

var (
	at  = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func TestAwardGrantsWhatTheStatsEarn(t *testing.T) {
	store := inmemory_player_store.New()

	require.NoError(t, players.NewTitleBook(store, catalog).Award(t.Context(), ada, players.Stats{TilesTaken: 3}, at))

	held, err := store.Titles(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, players.TitleIDs{"first", "third"}, held)
}

type refusingGrants struct {
	*inmemory_player_store.Store
}

func (refusingGrants) GrantTitles(context.Context, players.Grants, time.Time) error {
	return errors.New("nothing was to be granted")
}

func TestAwardWritesNothingWhenEveryEarnedTitleIsHeld(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.GrantTitles(t.Context(), players.Grants{ada: {"first"}}, at))

	err := players.NewTitleBook(refusingGrants{store}, catalog).Award(t.Context(), ada, players.Stats{TilesTaken: 2}, at)

	assert.NoError(t, err)
}

func TestTheTitlesOfAnAccountAreTheCatalogsObjects(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.GrantTitles(t.Context(), players.Grants{ada: {"third", "first"}}, at))

	titles, err := players.NewTitleBook(store, catalog).TitlesOf(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, []players.Title{first, third}, titles)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))
	book := players.NewTitleBook(store, catalog)

	_, err := book.TitlesOf(t.Context(), ada)
	require.Error(t, err)
	assert.Error(t, book.Award(t.Context(), ada, players.Stats{TilesTaken: 3}, at))
}
