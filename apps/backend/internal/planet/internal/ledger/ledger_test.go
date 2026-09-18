package ledger_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) { return o[tile], true }

// board plays takes onto one map and remembers them in order, the way Recording and a storage would.
type board struct {
	owners  owners
	takings []ledger.Taking
}

func newBoard(initial owners) *board {
	return &board{owners: initial}
}

func (b *board) take(tile uint32, scope, country string) {
	b.takings = append(b.takings, ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: b.owners[tile]})
	b.owners[tile] = country
}

func (b *board) restorations(scope string) []clicks.Restoration {
	runs := ledger.NewRuns(ledger.Caller{Scope: scope})
	for _, taking := range b.takings {
		runs.See(taking)
	}
	return runs.Restorations(b.owners)
}

func TestARevertGoesBackPastEveryTakeOfTheScopesRun(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "A", "fr")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "fr", To: "il"}}, b.restorations("A"))
}

func TestATakeBySomebodyElseStartsANewRun(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "B", "de")
	b.take(7, "A", "ps")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "de"}}, b.restorations("A"),
		"B's retake stands: only A's latest run is undone")
	assert.Empty(t, b.restorations("B"), "B no longer holds the tile")
}

func TestAScopePaintedOverHoldsNothingToRevert(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "B", "il")

	assert.Empty(t, b.restorations("A"))
	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "il", To: "ps"}}, b.restorations("B"))
}

func TestTwoScopesPaintingTheSameFlagDoNotHoldEachOthersTiles(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "B", "il")
	b.take(7, "C", "ps")

	assert.Empty(t, b.restorations("A"), "the tile wears A's flag, but C painted it")
	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}}, b.restorations("C"))
}

func TestAChangeTheLedgerNeverSawBreaksTheRun(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.owners[7] = ""
	b.take(7, "A", "ps")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: ""}}, b.restorations("A"),
		"the bomb gave the tile to nobody, so the run starts after it")
}

func TestATileBombedAfterTheLastTakeIsNotHeld(t *testing.T) {
	b := newBoard(owners{7: "il", 8: "il"})
	b.take(7, "A", "ps")
	b.take(8, "A", "ps")
	b.owners[8] = ""

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}}, b.restorations("A"))
}

func TestARunThatEndsWhereItStartedGivesNothingBack(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "A", "il")

	assert.Empty(t, b.restorations("A"))
}

func TestRunsCountEveryTileTheScopeTookHeldOrNot(t *testing.T) {
	b := newBoard(owners{})
	b.take(1, "A", "ps")
	b.take(2, "A", "ps")
	b.take(2, "B", "il")
	b.take(1, "A", "fr")

	runs := ledger.NewRuns(ledger.Caller{Scope: "A"})
	for _, taking := range b.takings {
		runs.See(taking)
	}

	assert.Equal(t, 2, runs.Touched())
	assert.Len(t, runs.Restorations(b.owners), 1)
}

func tally(takings []ledger.Taking, current owners, counts func(ledger.Taking) bool) []ledger.Player {
	t := ledger.NewTally(counts)
	for _, taking := range takings {
		t.See(taking)
	}
	return t.Players(current)
}

func every(ledger.Taking) bool { return true }

func TestAPlayerCountsItsTakesAndTheTilesItStillHolds(t *testing.T) {
	current := owners{1: "ps", 2: "il", 3: ""}
	players := tally([]ledger.Taking{
		{Tile: 1, Scope: "bot", Country: "ps", At: start},
		{Tile: 2, Scope: "bot", Country: "ps", At: start.Add(time.Second)},
		{Tile: 2, Scope: "defender", Country: "il", At: start.Add(2 * time.Second)},
		{Tile: 2, Scope: "bot", Country: "ps", At: start.Add(3 * time.Second)},
		{Tile: 2, Scope: "defender", Country: "il", At: start.Add(4 * time.Second)},
		{Tile: 3, Scope: "bot", Country: "ps", At: start.Add(5 * time.Second)},
	}, current, every)

	assert.Equal(t, []ledger.Player{
		{Scope: "bot", Tiles: 1, Takes: 4, FirstAt: start, LastAt: start.Add(5 * time.Second)},
		{Scope: "defender", Tiles: 1, Takes: 2, FirstAt: start.Add(2 * time.Second), LastAt: start.Add(4 * time.Second)},
	}, players, "a tile painted over and a tile bombed still count as takes, not as holds")
}

func TestAPlayerPaintedOverEverywhereStillShows(t *testing.T) {
	players := tally([]ledger.Taking{
		{Tile: 1, Scope: "bot", Country: "ps", At: start},
		{Tile: 1, Scope: "defender", Country: "il", At: start.Add(time.Second)},
	}, owners{1: "il"}, func(taking ledger.Taking) bool { return taking.Country == "ps" })

	assert.Equal(t, []ledger.Player{{Scope: "bot", Takes: 1, FirstAt: start, LastAt: start}}, players)
}

func TestATakeThatDoesNotCountEndsTheHoldOfOneThatDid(t *testing.T) {
	players := tally([]ledger.Taking{
		{Tile: 1, Scope: "bot", Country: "ps", At: start},
		{Tile: 1, Scope: "bot", Country: "fr", At: start.Add(time.Second)},
		{Tile: 1, Scope: "other", Country: "ps", At: start.Add(2 * time.Second)},
	}, owners{1: "ps"}, func(taking ledger.Taking) bool { return taking.Scope == "bot" })

	require.Len(t, players, 1)
	assert.Zero(t, players[0].Tiles, "the tile wears ps, but another scope painted it last")
	assert.Equal(t, 2, players[0].Takes)
}

func TestPlayersAreLatestFirstAndEqualTimesFallBackToScope(t *testing.T) {
	players := tally([]ledger.Taking{
		{Tile: 1, Scope: "bot", At: start.Add(2 * time.Second)},
		{Tile: 2, Scope: "player", At: start.Add(time.Second)},
		{Tile: 3, Scope: "also", At: start.Add(time.Second)},
	}, owners{}, every)

	assert.Equal(t, []string{"bot", "also", "player"}, scopes(players))
}

func TestByTakesPutsTheMostTakesFirstThenTheMostTilesHeld(t *testing.T) {
	players := ledger.ByTakes([]ledger.Player{
		{Scope: "latest", Takes: 1},
		{Scope: "bot", Takes: 50},
		{Scope: "holder", Takes: 5, Tiles: 5},
		{Scope: "older", Takes: 1},
		{Scope: "painted over", Takes: 5},
	})

	assert.Equal(t, []string{"bot", "holder", "painted over", "latest", "older"}, scopes(players))
}

func scopes(players []ledger.Player) []string {
	out := make([]string, 0, len(players))
	for _, player := range players {
		out = append(out, player.Scope)
	}
	return out
}

func TestTopCutsToTheLimitOrTheDefault(t *testing.T) {
	players := make([]ledger.Player, 30)

	assert.Len(t, ledger.Top(players, 3), 3)
	assert.Len(t, ledger.Top(players, 0), 20)
	assert.Len(t, ledger.Top(players[:2], 5), 2)
}

func TestAPlayersRatesAreItsCountsOverTheTimeFromFirstToLastTake(t *testing.T) {
	player := ledger.Player{Tiles: 30, Takes: 120, FirstAt: start, LastAt: start.Add(10 * time.Minute)}

	assert.Equal(t, 10*time.Minute, player.ActiveFor())
	assert.InDelta(t, 3.0, player.TilesPerMinute(), 1e-9)
	assert.InDelta(t, 12.0, player.TakesPerMinute(), 1e-9)

	single := ledger.Player{Tiles: 1, Takes: 1, FirstAt: start, LastAt: start}
	assert.Zero(t, single.TilesPerMinute(), "one instant has no rate")
	assert.Zero(t, single.TakesPerMinute())
}

func TestAServingPlayerCarriesTheSentence(t *testing.T) {
	player := ledger.Player{Scope: "bot"}
	player.Serving(antibot.Sentence{Offence: 2, Until: start})

	assert.Equal(t, ledger.Player{Scope: "bot", Banned: true, BannedUntil: start, Offence: 2}, player)
}

func replay(storage ledger.Storage) []ledger.Taking {
	var takings []ledger.Taking
	storage.Replay(func(taking ledger.Taking) { takings = append(takings, taking) })
	return takings
}

func TestRetentionForgetsTakesPastIt(t *testing.T) {
	clock := cptime.NewFixedClock(start)
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	takings.Append(ledger.Taking{Tile: 1, Scope: "1.2.3.4", At: start})
	takings.Append(ledger.Taking{Tile: 1, Scope: "1.2.3.4", At: start.Add(30 * time.Minute)})

	clock.Advance(61 * time.Minute)
	ledger.NewRetention(ledger.Config{Retention: time.Hour}, takings, clock).Sweep()

	remaining := replay(takings)
	require.Len(t, remaining, 1)
	assert.Equal(t, start.Add(30*time.Minute), remaining[0].At)
}

type stubTiles struct {
	owners map[uint32]string
	err    error
}

func (s *stubTiles) Owner(tile uint32) (string, bool) { return s.owners[tile], true }

func (s *stubTiles) Set(_ context.Context, tile uint32, value string) error {
	if s.err != nil {
		return s.err
	}
	s.owners[tile] = value
	return nil
}

func TestRecordingNotesTheCallersScopeAndOnlyAChange(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{1: "de", 2: "fr"}}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	recording := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start))

	ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::1")

	require.NoError(t, recording.Set(ctx, 1, "fr"))
	require.NoError(t, recording.Set(ctx, 2, "fr"))

	assert.Equal(t, []ledger.Taking{{Tile: 1, Scope: "2001:db8::/64", Country: "fr", Previous: "de", At: start}},
		replay(takings), "a v6 caller is its /64, and a tile it already held is no take")
}

func TestRecordingKeepsEveryTake(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{7: "de"}}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	recording := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start))

	require.NoError(t, recording.Set(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 7, "fr"))
	require.NoError(t, recording.Set(cpctx.AddIPToContext(t.Context(), "5.6.7.8"), 7, "de"))
	require.NoError(t, recording.Set(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 7, "ps"))

	assert.Equal(t, []ledger.Taking{
		{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start},
		{Tile: 7, Scope: "5.6.7.8", Country: "de", Previous: "fr", At: start},
		{Tile: 7, Scope: "1.2.3.4", Country: "ps", Previous: "de", At: start},
	}, replay(takings))
}

func TestRecordingNotesNothingForAFailedWrite(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{}, err: errors.New("out of range")}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))

	err := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start)).Set(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 1, "fr")
	require.ErrorIs(t, err, tiles.err)

	assert.Empty(t, replay(takings))
}

func TestRecordingNotesTheAccountTheTokenNamed(t *testing.T) {
	tiles := &stubTiles{owners: map[uint32]string{1: "de"}}
	takings := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	recording := ledger.NewRecording(tiles, takings, cptime.NewFixedClock(start))

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
	require.NoError(t, recording.Set(ctx, 1, "fr"))

	assert.Equal(t, []ledger.Taking{{Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Previous: "de", At: start}},
		replay(takings))
}

func TestEachAccountOnOneScopeIsAPlayerOfItsOwn(t *testing.T) {
	players := tally([]ledger.Taking{
		{Tile: 1, Scope: "campus", Account: "alice", Country: "ps", At: start},
		{Tile: 2, Scope: "campus", Account: "bob", Country: "ps", At: start},
		{Tile: 3, Scope: "campus", Account: "bob", Country: "ps", At: start.Add(time.Second)},
		{Tile: 4, Scope: "campus", Country: "ps", At: start.Add(time.Second)},
	}, owners{1: "ps", 2: "ps", 3: "ps", 4: "ps"}, every)

	assert.Equal(t, []ledger.Player{
		{Scope: "campus", Tiles: 1, Takes: 1, FirstAt: start.Add(time.Second), LastAt: start.Add(time.Second)},
		{Scope: "campus", Account: "bob", Tiles: 2, Takes: 2, FirstAt: start, LastAt: start.Add(time.Second)},
		{Scope: "campus", Account: "alice", Tiles: 1, Takes: 1, FirstAt: start, LastAt: start},
	}, players)
}

func TestARevertOfAnAccountFollowsItAcrossScopes(t *testing.T) {
	b := newBoard(owners{7: "il", 8: "il"})
	b.takings = append(b.takings,
		ledger.Taking{Tile: 7, Scope: "campus", Account: "guest", Country: "ps", Previous: "il"},
		ledger.Taking{Tile: 7, Scope: "home", Account: "guest", Country: "fr", Previous: "ps"},
		ledger.Taking{Tile: 8, Scope: "home", Account: "other", Country: "ps", Previous: "il"},
	)
	b.owners[7], b.owners[8] = "fr", "ps"

	runs := ledger.NewRuns(ledger.Caller{Account: "guest"})
	for _, taking := range b.takings {
		runs.See(taking)
	}

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "fr", To: "il"}}, runs.Restorations(b.owners))
}

func TestParseCallerTakesAScopeOrAnAccount(t *testing.T) {
	caller, err := ledger.ParseCaller("2001:db8::9", "")
	require.NoError(t, err)
	assert.Equal(t, ledger.Caller{Scope: "2001:db8::/64"}, caller)

	caller, err = ledger.ParseCaller("", "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10")
	require.NoError(t, err)
	assert.Equal(t, ledger.Caller{Account: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}, caller)

	_, err = ledger.ParseCaller("", "")
	require.ErrorIs(t, err, ledger.ErrNoCaller)
	_, err = ledger.ParseCaller("1.2.3.4", "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
	require.ErrorIs(t, err, ledger.ErrNoCaller)
	_, err = ledger.ParseCaller("bot", "")
	require.ErrorIs(t, err, ledger.ErrInvalidScope)
	_, err = ledger.ParseCaller("", "guest")
	require.ErrorIs(t, err, ledger.ErrInvalidAccount)
}
