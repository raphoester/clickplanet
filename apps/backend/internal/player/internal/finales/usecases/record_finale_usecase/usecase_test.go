package record_finale_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/usecases/record_finale_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/inmemory_podium_store"
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
	guest     = players.AccountID{15: 4}
	seasonEnd = record_finale_usecase.In{Season: 0, Book: "season-0", Finale: "season-0/finale"}
)

type stubScoring struct {
	table    []finales.Place
	fighters []finales.Fighter
	err      error
}

func (s stubScoring) Table(context.Context, finales.BookID) ([]finales.Place, error) {
	return s.table, s.err
}

func (s stubScoring) Fighters(context.Context, finales.RoundID) ([]finales.Fighter, error) {
	return s.fighters, s.err
}

type fixture struct {
	stats    *inmemory_player_store.Store
	titles   *inmemory_title_store.Store
	accounts *titles.FakeAccounts
	fights   *inmemory_fight_store.Store
}

func setUp(t *testing.T) fixture {
	t.Helper()

	f := fixture{
		stats: inmemory_player_store.New(), titles: inmemory_title_store.New(),
		accounts: titles.NewFakeAccounts(), fights: inmemory_fight_store.New(),
	}
	for i, account := range []players.AccountID{ada, bob, carol, guest} {
		require.NoError(t, f.stats.Record(t.Context(), account, players.TakeAt(uint64(i), 0, november)))
		f.accounts.Create(account, players.AccountOf(account != guest, november))
	}
	return f
}

func (f fixture) record(t *testing.T, scoring stubScoring) error {
	t.Helper()

	award := award_titles_usecase.New(
		f.stats, titles.NewCareers(f.accounts, f.fights, inmemory_podium_store.New()), titles.NewBook(f.titles, titles.NewCatalog()), cptime.NewFixedClock(end),
	)
	return record_finale_usecase.New(scoring, f.fights, award).Execute(t.Context(), seasonEnd) //nolint:wrapcheck // the tests read the use case's error.
}

func (f fixture) held(t *testing.T, account players.AccountID) titles.IDs {
	t.Helper()

	held, err := f.titles.Held(t.Context(), account)
	require.NoError(t, err)
	return held
}

var finalNight = stubScoring{
	table: []finales.Place{{Country: "fr", Rank: 1}, {Country: "de", Rank: 2}, {Country: "it", Rank: 3}, {Country: "es", Rank: 4}},
	fighters: []finales.Fighter{
		{Account: ada, Country: "fr", Tiles: 50},
		{Account: bob, Country: "es", Tiles: 80},
		{Account: carol, Country: "tv", Tiles: 50},
		{Account: guest, Country: "fr", Tiles: 200},
	},
}

func TestEveryFighterIsRecordedUnderItsFlagsPlace(t *testing.T) {
	f := setUp(t)

	require.NoError(t, f.record(t, finalNight))

	found, err := f.fights.Fights(t.Context(), []players.AccountID{ada, bob, carol, guest})
	require.NoError(t, err)
	snapshots := map[players.AccountID]finales.FightSnapshot{}
	for account, fights := range found {
		require.Len(t, fights, 1)
		snapshots[account] = fights[0].TakeSnapshot()
	}
	assert.Equal(t, map[players.AccountID]finales.FightSnapshot{
		ada:   {Account: ada, Season: 0, Country: "fr", Rank: 1, TilesRank: 3},
		bob:   {Account: bob, Season: 0, Country: "es", Rank: 4, TilesRank: 2},
		carol: {Account: carol, Season: 0, Country: "tv", Rank: 0, TilesRank: 3},
		guest: {Account: guest, Season: 0, Country: "fr", Rank: 1, TilesRank: 1},
	}, snapshots, "a flag off the table has no place, and a guest's fight is kept for when it signs in")
}

func TestEachSignedInFighterIsAwardedItsTitlesAndAGuestNone(t *testing.T) {
	f := setUp(t)

	require.NoError(t, f.record(t, finalNight))

	assert.Equal(t, titles.IDs{"season_0_gold", "season_0_battle_top_3", "season_0_veteran"}, f.held(t, ada))
	assert.Equal(t, titles.IDs{"season_0_battle_top_2", "season_0_veteran"}, f.held(t, bob))
	assert.Equal(t, titles.IDs{"season_0_battle_top_3", "season_0_veteran"}, f.held(t, carol))
	assert.Empty(t, f.held(t, guest), "first by tiles, but a guest earns nothing yet")
}

func TestTheSeasonToldAgainChangesNothing(t *testing.T) {
	f := setUp(t)
	require.NoError(t, f.record(t, finalNight))

	require.NoError(t, f.record(t, finalNight))

	assert.Equal(t, titles.IDs{"season_0_gold", "season_0_battle_top_3", "season_0_veteran"}, f.held(t, ada))
	assert.Equal(t, titles.IDs{"season_0_battle_top_2", "season_0_veteran"}, f.held(t, bob))
}

func TestTheSeasonToldAgainRanksTheFightsRecordedBeforeTheRankWasKept(t *testing.T) {
	f := setUp(t)
	before, err := finales.FightSnapshot{Account: bob, Season: 0, Country: "es", Rank: 4}.Restore()
	require.NoError(t, err)
	require.NoError(t, f.fights.Record(t.Context(), []finales.Fight{before}))

	require.NoError(t, f.record(t, finalNight))

	found, err := f.fights.Fights(t.Context(), []players.AccountID{bob})
	require.NoError(t, err)
	assert.Equal(t, uint32(2), found[bob][0].TakeSnapshot().TilesRank)
	assert.Equal(t, titles.IDs{"season_0_battle_top_2", "season_0_veteran"}, f.held(t, bob))
}

func TestAScoringFailureRecordsNothing(t *testing.T) {
	f := setUp(t)

	require.Error(t, f.record(t, stubScoring{err: errors.New("scoring is down")}))

	found, err := f.fights.Fights(t.Context(), []players.AccountID{ada})
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestAFighterWithNoFlagIsAnErrorAndRecordsNothing(t *testing.T) {
	f := setUp(t)

	err := f.record(t, stubScoring{fighters: []finales.Fighter{{Account: ada, Country: "fr"}, {Account: bob}}})

	require.ErrorIs(t, err, finales.ErrNoCountry)
	assert.Empty(t, f.held(t, ada))
}

func TestAFailedRecordAwardsNothing(t *testing.T) {
	f := setUp(t)
	f.fights.FailWith(errors.New("postgres is down"))

	require.Error(t, f.record(t, finalNight))

	assert.Empty(t, f.held(t, ada))
}
