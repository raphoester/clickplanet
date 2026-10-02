package backfill_titles_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/backfill_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	first = players.FakeTitle{Key: "first", Tiles: 1}
	third = players.FakeTitle{Key: "third", Tiles: 3}
)

func account(i int) players.AccountID {
	return players.AccountID{14: byte(i >> 8), 15: byte(i)}
}

func take(t *testing.T, store *inmemory_player_store.Store, i, tiles int) {
	t.Helper()

	for range tiles {
		require.NoError(t, store.RecordTake(t.Context(), account(i), now))
	}
}

func held(t *testing.T, store *inmemory_player_store.Store, i int) players.TitleIDs {
	t.Helper()

	titles, err := store.Titles(t.Context(), account(i))
	require.NoError(t, err)
	return titles
}

func backfill(t *testing.T, titles backfill_titles_usecase.Titles, store *inmemory_player_store.Store, catalog players.Catalog) (backfill_titles_usecase.Backfill, error) {
	t.Helper()

	return backfill_titles_usecase.New(store, titles, catalog, cptime.NewFixedClock(now)).Execute(t.Context()) //nolint:wrapcheck // the tests read the use case's error.
}

func TestEveryAccountGetsTheTitlesItsStatsEarnAcrossPages(t *testing.T) {
	store := inmemory_player_store.New()
	for i := 1; i <= 1_001; i++ {
		take(t, store, i, 1)
	}
	take(t, store, 1_001, 2)

	done, err := backfill(t, store, store, players.Catalog{first, third})

	require.NoError(t, err)
	assert.Equal(t, backfill_titles_usecase.Backfill{Titles: players.TitleIDs{"first", "third"}, Accounts: 1_001}, done)
	assert.Equal(t, players.TitleIDs{"first"}, held(t, store, 1))
	assert.Equal(t, players.TitleIDs{"first"}, held(t, store, 500))
	assert.Equal(t, players.TitleIDs{"first", "third"}, held(t, store, 1_001))
}

func TestABackfilledTitleIsNotBackfilledAgain(t *testing.T) {
	store := inmemory_player_store.New()
	take(t, store, 1, 1)
	_, err := backfill(t, store, store, players.Catalog{first})
	require.NoError(t, err)

	done, err := backfill(t, refusingGrants{store}, store, players.Catalog{first})

	require.NoError(t, err)
	assert.Empty(t, done.Titles)
}

func TestATitleAddedLaterIsBackfilledOnItsOwn(t *testing.T) {
	store := inmemory_player_store.New()
	take(t, store, 1, 3)
	_, err := backfill(t, store, store, players.Catalog{first})
	require.NoError(t, err)

	done, err := backfill(t, store, store, players.Catalog{first, third})

	require.NoError(t, err)
	assert.Equal(t, players.TitleIDs{"third"}, done.Titles)
	assert.Equal(t, players.TitleIDs{"first", "third"}, held(t, store, 1))
}

type refusingGrants struct {
	*inmemory_player_store.Store
}

func (refusingGrants) GrantTitles(context.Context, players.Grants, time.Time) error {
	return errors.New("postgres is down")
}

func TestAFailedBackfillIsTriedAgainNextTime(t *testing.T) {
	store := inmemory_player_store.New()
	take(t, store, 1, 1)

	_, err := backfill(t, refusingGrants{store}, store, players.Catalog{first})
	require.Error(t, err)

	backfilled, err := store.BackfilledTitles(t.Context())
	require.NoError(t, err)
	assert.Empty(t, backfilled)
}
