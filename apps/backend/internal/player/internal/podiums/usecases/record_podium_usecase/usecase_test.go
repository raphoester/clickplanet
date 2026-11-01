package record_podium_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/inmemory_podium_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/usecases/record_podium_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	end       = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	november  = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	ada       = players.AccountID{15: 1}
	bob       = players.AccountID{15: 2}
	carol     = players.AccountID{15: 3}
	dan       = players.AccountID{15: 4}
	seasonEnd = record_podium_usecase.In{Season: 0, Book: "season-0"}
)

type stubScoring struct {
	board []podiums.Entry
	err   error
}

func (s stubScoring) Board(context.Context, finales.BookID) ([]podiums.Entry, error) {
	return s.board, s.err
}

type fixture struct {
	stats    *inmemory_player_store.Store
	titles   *inmemory_title_store.Store
	accounts *titles.FakeAccounts
	places   *inmemory_podium_store.Store
}

func setUp(t *testing.T) fixture {
	t.Helper()

	f := fixture{
		stats: inmemory_player_store.New(), titles: inmemory_title_store.New(),
		accounts: titles.NewFakeAccounts(), places: inmemory_podium_store.New(),
	}
	for i, account := range []players.AccountID{ada, bob, carol, dan} {
		require.NoError(t, f.stats.Record(t.Context(), account, players.TakeAt(uint64(i), 0, november)))
		f.accounts.Create(account, players.AccountOf(true, november))
	}
	return f
}

func (f fixture) record(t *testing.T, scoring stubScoring) error {
	t.Helper()

	award := award_titles_usecase.New(
		f.stats, titles.NewCareers(f.accounts, inmemory_fight_store.New(), f.places),
		titles.NewBook(f.titles, titles.NewCatalog()), cptime.NewFixedClock(end),
	)
	return record_podium_usecase.New(scoring, f.places, award).Execute(t.Context(), seasonEnd) //nolint:wrapcheck // the tests read the use case's error.
}

func (f fixture) held(t *testing.T, account players.AccountID) titles.IDs {
	t.Helper()

	held, err := f.titles.Held(t.Context(), account)
	require.NoError(t, err)
	return held
}

var board = stubScoring{board: []podiums.Entry{
	{Account: ada, Rank: 1}, {Account: bob, Rank: 2}, {Account: carol, Rank: 2}, {Account: dan, Rank: 4},
}}

func TestTheFirstThreeStepsOfTheBoardAreItsPodiumAndATieSharesAStep(t *testing.T) {
	f := setUp(t)

	require.NoError(t, f.record(t, board))

	found, err := f.places.Places(t.Context(), []players.AccountID{ada, bob, carol, dan})
	require.NoError(t, err)
	ranks := map[players.AccountID]uint32{}
	for account, places := range found {
		require.Len(t, places, 1)
		ranks[account] = places[0].TakeSnapshot().Rank
	}
	assert.Equal(t, map[players.AccountID]uint32{ada: 1, bob: 2, carol: 2}, ranks, "4th is off the podium")
}

func TestEachPlayerOnThePodiumIsAwardedItsTop(t *testing.T) {
	f := setUp(t)

	require.NoError(t, f.record(t, board))

	assert.Equal(t, titles.IDs{"season_0_top_1"}, f.held(t, ada))
	assert.Equal(t, titles.IDs{"season_0_top_2"}, f.held(t, bob))
	assert.Equal(t, titles.IDs{"season_0_top_2"}, f.held(t, carol))
	assert.Empty(t, f.held(t, dan))
}

func TestTheSeasonToldAgainChangesNothing(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.record(t, board))

	require.NoError(t, f.record(t, board))

	assert.Equal(t, titles.IDs{"season_0_top_1"}, f.held(t, ada))
}

func TestAScoringFailureRecordsNothing(t *testing.T) {
	f := setUp(t)

	require.Error(t, f.record(t, stubScoring{err: errors.New("scoring is down")}))

	found, err := f.places.Places(t.Context(), []players.AccountID{ada})
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestAFailedRecordAwardsNothing(t *testing.T) {
	f := setUp(t)
	f.places.FailWith(errors.New("postgres is down"))

	require.Error(t, f.record(t, board))

	assert.Empty(t, f.held(t, ada))
}
