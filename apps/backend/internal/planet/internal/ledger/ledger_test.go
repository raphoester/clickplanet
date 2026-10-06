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

type board struct {
	owners owners
	events []ledger.Event
}

func newBoard(initial owners) *board {
	return &board{owners: initial}
}

func (b *board) take(tile uint32, scope, country string) {
	b.events = append(b.events, ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: b.owners[tile]})
	b.owners[tile] = country
}

func (b *board) restorations(scope string) []clicks.Restoration {
	runs := ledger.NewRuns(ledger.Caller{Scope: scope})
	for _, event := range b.events {
		runs.See(event)
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
		"a revert gave the tile to nobody, so the run starts after it")
}

func TestATileEmptiedAfterTheLastTakeIsNotHeld(t *testing.T) {
	b := newBoard(owners{7: "il", 8: "il"})
	b.take(7, "A", "ps")
	b.take(8, "A", "ps")
	b.owners[8] = ""

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}}, b.restorations("A"))
}

func (b *board) bomb(scope string, tiles ...uint32) {
	blast := clicks.Blast{Cleared: tiles}
	for _, tile := range tiles {
		blast.Owners = append(blast.Owners, b.owners[tile])
		b.owners[tile] = ""
	}
	b.events = append(b.events, ledger.Bombing{Scope: scope, Blast: blast})
}

func TestARevertGivesTheBombedTilesBackToTheirOwners(t *testing.T) {
	b := newBoard(owners{7: "il", 8: "de"})
	b.bomb("A", 7, 8)

	assert.Equal(t, []clicks.Restoration{
		{Tile: 7, From: "", To: "il"},
		{Tile: 8, From: "", To: "de"},
	}, b.restorations("A"))
}

func TestARevertOfABomberGoesBackPastItsOwnTakes(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.bomb("A", 7)

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "", To: "il"}}, b.restorations("A"))
}

func TestABombedTileRetakenStaysWithTheTaker(t *testing.T) {
	b := newBoard(owners{7: "il", 8: "de"})
	b.bomb("A", 7, 8)
	b.take(8, "B", "fr")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "", To: "il"}}, b.restorations("A"))
}

func TestABombBreaksTheRunOfTheScopeItHit(t *testing.T) {
	b := newBoard(owners{7: "il"})
	b.take(7, "A", "ps")
	b.bomb("B", 7)
	b.take(7, "A", "ps")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: ""}}, b.restorations("A"))
}

func changes(event ledger.Event) []ledger.Taking {
	var takings []ledger.Taking
	event.Replay(func(taking ledger.Taking) { takings = append(takings, taking) })
	return takings
}

func TestABombingReplaysAsOneClearPerTileItHit(t *testing.T) {
	bombing := ledger.Bombing{Scope: "A", Account: "guest", At: start, Blast: clicks.Blast{
		Tile: 8, CountryID: "fr", Cleared: []uint32{7, 8}, Owners: []string{"il", "de"},
	}}

	assert.Equal(t, []ledger.Taking{
		{Tile: 7, Scope: "A", Account: "guest", Previous: "il", At: start},
		{Tile: 8, Scope: "A", Account: "guest", Previous: "de", At: start},
	}, changes(bombing))
	assert.Empty(t, changes(ledger.Bombing{Scope: "A", At: start, Blast: clicks.Blast{CountryID: "fr"}}), "a bomb in the sea")
}

func TestASpreadReplaysAsOneTakePerTileItTook(t *testing.T) {
	spreading := ledger.Spreading{Tile: 1, Scope: "A", Account: "guest", Country: "fr", At: start, Impacts: []clicks.Impact{
		{Tile: 2, Owner: "de", Outcome: clicks.Taken},
		{Tile: 3, Owner: "pl", Outcome: clicks.Shielded, Shields: 4},
	}}

	assert.Equal(t, []ledger.Taking{{Tile: 2, Scope: "A", Account: "guest", Country: "fr", Previous: "de", At: start}},
		changes(spreading))
	assert.Equal(t, changes(ledger.Enclosing(spreading)), changes(spreading), "an enclosure replays the same way")
}

func TestAStrikeAndAShieldReplayAsNothing(t *testing.T) {
	assert.Empty(t, changes(ledger.Striking{Tile: 3, Scope: "A", Country: "fr", Owner: "pl", At: start}))
	assert.Empty(t, changes(ledger.Shielding{Tile: 3, Scope: "A", Country: "pl", Shields: 1, At: start}))
}

func TestATakingReplaysAsItself(t *testing.T) {
	taking := ledger.Taking{Tile: 7, Scope: "A", Country: "fr", Previous: "il", At: start}

	assert.Equal(t, []ledger.Taking{taking}, changes(taking))
}

func TestABombingIsWrittenDownOnTheTileItHit(t *testing.T) {
	entry, err := ledger.Bombing{Scope: "A", Account: "guest", At: start, Blast: clicks.Blast{
		Tile: 8, CountryID: "fr", Point: clicks.Vec3{X: 0.6, Z: 0.8}, Radius: 0.032,
		Cleared: []uint32{7, 8, 9}, Owners: []string{"il", "de", "il"}, Struck: []uint32{10}, Left: []int{0},
	}}.Entry()
	require.NoError(t, err)

	payload := entry.Payload
	entry.Payload = nil
	assert.Equal(t, ledger.Entry{Kind: "bomb", Tile: 8, Scope: "A", Account: "guest", Previous: "de", At: start}, entry,
		"the tile it hit was emptied")
	assert.JSONEq(t, `{"flag": "fr", "point": [0.6, 0, 0.8], "radius": 0.032,
		"cleared": [{"tile": 7, "owner": "il"}, {"tile": 8, "owner": "de"}, {"tile": 9, "owner": "il"}],
		"struck": [{"tile": 10, "shields": 0}]}`, string(payload))
}

func TestASpreadIsWrittenDownOnTheTileClickedWithWhatItTookAndStruck(t *testing.T) {
	entry, err := ledger.Spreading{Tile: 1, Scope: "A", Account: "guest", Country: "fr", At: start, Impacts: []clicks.Impact{
		{Tile: 2, Owner: "de", Outcome: clicks.Taken},
		{Tile: 3, Owner: "pl", Outcome: clicks.Shielded, Shields: 4},
		{Tile: 5, Outcome: clicks.Taken},
	}}.Entry()
	require.NoError(t, err)

	payload := entry.Payload
	entry.Payload = nil
	assert.Equal(t, ledger.Entry{Kind: "spread", Tile: 1, Scope: "A", Account: "guest", Country: "fr", At: start}, entry)
	assert.JSONEq(t, `{"taken": [{"tile": 2, "owner": "de"}, {"tile": 5}], "struck": [{"tile": 3, "owner": "pl", "shields": 4}]}`,
		string(payload))
}

func TestAStrikeIsWrittenDownOnTheTileWithItsOwnerAndTheShieldsLeft(t *testing.T) {
	entry, err := ledger.Striking{Tile: 3, Scope: "A", Country: "fr", Owner: "pl", Shields: 0, At: start}.Entry()
	require.NoError(t, err)

	assert.Equal(t, ledger.Entry{
		Kind: "strike", Tile: 3, Scope: "A", Country: "fr", Previous: "pl", At: start, Payload: []byte(`{"shields":0}`),
	}, entry)
}

func TestEveryKindReadsBackAsItWasWrittenDown(t *testing.T) {
	for _, event := range []ledger.Event{
		ledger.Taking{Tile: 7, Scope: "A", Account: "guest", Country: "fr", Previous: "il", At: start},
		ledger.Taking{Tile: 7, Scope: "A", Previous: "pl", At: start},
		ledger.Bombing{Scope: "A", Account: "guest", At: start, Blast: clicks.Blast{
			Tile: 8, CountryID: "fr", Point: clicks.Vec3{X: 0.6, Z: 0.8}, Radius: 0.032,
			Cleared: []uint32{7, 8, 9}, Owners: []string{"il", "de", "il"},
		}},
		ledger.Bombing{Scope: "A", At: start, Blast: clicks.Blast{CountryID: "fr", Point: clicks.Vec3{Y: 1}, Radius: 0.032}},
		ledger.Bombing{Scope: "A", At: start, Blast: clicks.Blast{
			Tile: 8, CountryID: "fr", Cleared: []uint32{7}, Owners: []string{"il"}, Struck: []uint32{8, 9}, Left: []int{2, 0},
		}},
		ledger.Striking{Tile: 3, Scope: "A", Account: "guest", Country: "fr", Owner: "pl", Shields: 2, At: start},
		ledger.Spreading{Tile: 1, Scope: "A", Account: "guest", Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "de", Outcome: clicks.Taken},
			{Tile: 3, Owner: "pl", Outcome: clicks.Shielded, Shields: 4},
			{Tile: 5, Outcome: clicks.Taken},
		}},
		ledger.Spreading{Tile: 7, Scope: "A", Country: "fr", At: start},
		ledger.Enclosing{Tile: 1, Scope: "A", Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 8, Owner: "de", Outcome: clicks.Taken},
			{Tile: 9, Owner: "de", Outcome: clicks.Shielded},
		}},
		ledger.Shielding{Tile: 4, Scope: "A", Account: "guest", Country: "fr", Shields: 10, At: start},
	} {
		entry, err := event.Entry()
		require.NoError(t, err)

		read, err := ledger.EventOf(entry)
		require.NoError(t, err)
		assert.Equal(t, event, read)
	}
}

func TestAnEntryOfAKindTheLedgerDoesNotKnowReadsBackAsNothing(t *testing.T) {
	_, err := ledger.EventOf(ledger.Entry{Kind: "quake", Tile: 1})

	require.ErrorIs(t, err, ledger.ErrUnknownKind)
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
	for _, event := range b.events {
		runs.See(event)
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

func TestABombIsOneTakeThatHoldsNothingAndEndsTheHoldsOfTheTilesItHit(t *testing.T) {
	tally := ledger.NewTally(every)
	for _, event := range []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "player", Country: "ps", At: start},
		ledger.Taking{Tile: 2, Scope: "player", Country: "ps", At: start},
		ledger.Bombing{Scope: "bomber", At: start.Add(time.Second), Blast: clicks.Blast{
			Cleared: []uint32{1, 2}, Owners: []string{"ps", "ps"},
		}},
		ledger.Taking{Tile: 1, Scope: "other", Country: "ps", At: start.Add(2 * time.Second)},
	} {
		tally.See(event)
	}

	assert.Equal(t, []ledger.Player{
		{Scope: "other", Tiles: 1, Takes: 1, FirstAt: start.Add(2 * time.Second), LastAt: start.Add(2 * time.Second)},
		{Scope: "bomber", Takes: 1, FirstAt: start.Add(time.Second), LastAt: start.Add(time.Second)},
		{Scope: "player", Takes: 2, FirstAt: start, LastAt: start},
	}, tally.Players(owners{1: "ps", 2: ""}), "a bomb is one act, and the tiles it hit are nobody's")
}

func TestABombTakesNoTileForAnyFlag(t *testing.T) {
	tally := ledger.NewTally(func(taking ledger.Taking) bool { return taking.Country == "ps" })
	tally.See(ledger.Taking{Tile: 1, Scope: "player", Country: "ps", At: start})
	tally.See(ledger.Bombing{Scope: "bomber", At: start, Blast: clicks.Blast{Cleared: []uint32{1}, Owners: []string{"ps"}}})

	assert.Equal(t, []ledger.Player{{Scope: "player", Takes: 1, FirstAt: start, LastAt: start}},
		tally.Players(owners{1: ""}))
}

func TestASpreadIsOneTakeThatHoldsEveryTileItTookAndAStrikeIsNone(t *testing.T) {
	tally := ledger.NewTally(every)
	for _, event := range []ledger.Event{
		ledger.Taking{Tile: 1, Scope: "player", Country: "fr", At: start},
		ledger.Spreading{Tile: 1, Scope: "player", Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 2, Outcome: clicks.Taken},
			{Tile: 3, Outcome: clicks.Taken},
			{Tile: 4, Owner: "de", Outcome: clicks.Shielded},
		}},
		ledger.Striking{Tile: 4, Scope: "player", Country: "fr", Owner: "de", At: start.Add(time.Second)},
		ledger.Shielding{Tile: 1, Scope: "player", Country: "fr", Shields: 1, At: start.Add(time.Second)},
	} {
		tally.See(event)
	}

	assert.Equal(t, []ledger.Player{{Scope: "player", Tiles: 3, Takes: 2, FirstAt: start, LastAt: start}},
		tally.Players(owners{1: "fr", 2: "fr", 3: "fr", 4: "de"}), "the click and the spread are two acts")
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
	storage.Replay(func(event ledger.Event) { takings = append(takings, changes(event)...) })
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

var errOutOfRange = errors.New("out of range")

type stubTiles struct {
	owners  map[uint32]string
	shields map[uint32]int
}

func newTiles(owners map[uint32]string) *stubTiles {
	return &stubTiles{owners: owners, shields: map[uint32]int{}}
}

const lastTile = 100

func (s *stubTiles) Owner(tile uint32) (string, bool) { return s.owners[tile], true }

func (s *stubTiles) Shields(tile uint32) int { return s.shields[tile] }

func (s *stubTiles) Strike(_ context.Context, tile uint32, _ string) bool {
	if s.shields[tile] == 0 {
		return false
	}
	s.shields[tile]--
	return true
}

func (s *stubTiles) Set(_ context.Context, tile uint32, value string) error {
	if tile > lastTile {
		return errOutOfRange
	}
	s.owners[tile] = value
	return nil
}

func (s *stubTiles) Click(ctx context.Context, tile uint32, value string) error {
	return s.Set(ctx, tile, value)
}

func (s *stubTiles) Shield(_ context.Context, tile uint32, _ string, _ int) error {
	if tile > lastTile {
		return errOutOfRange
	}
	s.shields[tile]++
	return nil
}

func (s *stubTiles) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared := blast
	cleared.Cleared = nil
	for _, tile := range blast.Cleared {
		switch {
		case tile > lastTile:
			return clicks.Blast{}, errOutOfRange
		case s.shields[tile] > 0:
			s.shields[tile]--
			cleared.Struck = append(cleared.Struck, tile)
			cleared.Left = append(cleared.Left, s.shields[tile])
		case s.owners[tile] != "":
			cleared.Cleared = append(cleared.Cleared, tile)
			cleared.Owners = append(cleared.Owners, s.owners[tile])
			s.owners[tile] = ""
		}
	}
	return cleared, nil
}

func recorded(tiles *stubTiles) (ledger.Recording, *inmemory_ledger_storage.Storage) {
	events := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	return ledger.NewRecording(tiles, clicks.NewClaiming(tiles), events, cptime.NewFixedClock(start)), events
}

func written(storage ledger.Storage) []ledger.Event {
	var events []ledger.Event
	storage.Replay(func(event ledger.Event) { events = append(events, event) })
	return events
}

func caller(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

func TestRecordingNotesABombWithTheFlagEachTileWoreAndTheShieldsEachStruckTileKept(t *testing.T) {
	tiles := newTiles(map[uint32]string{1: "de", 2: "il", 4: "pl"})
	tiles.shields[4] = 2
	recording, events := recorded(tiles)

	blast, err := recording.Clear(caller(t), clicks.Blast{Tile: 2, CountryID: "fr", Cleared: []uint32{1, 2, 3, 4}})
	require.NoError(t, err)

	assert.Equal(t, []uint32{1, 2}, blast.Cleared)
	assert.Equal(t, map[uint32]string{1: "", 2: "", 4: "pl"}, tiles.owners)
	assert.Equal(t, []ledger.Event{ledger.Bombing{Scope: "1.2.3.4", Account: "a-guest", At: start, Blast: clicks.Blast{
		Tile: 2, CountryID: "fr", Cleared: []uint32{1, 2}, Owners: []string{"de", "il"}, Struck: []uint32{4}, Left: []int{1},
	}}}, written(events))
	assert.Equal(t, []ledger.Taking{
		{Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Previous: "de", At: start},
		{Tile: 2, Scope: "1.2.3.4", Account: "a-guest", Previous: "il", At: start},
	}, replay(events))
}

func TestRecordingNotesABombInTheSeaThatClearedNothing(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{}))

	_, err := recording.Clear(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), clicks.Blast{CountryID: "fr"})
	require.NoError(t, err)

	assert.Empty(t, replay(events), "a splash takes and clears no tile")
	assert.Len(t, written(events), 1, "but it is an event of its own")
}

func TestRecordingNotesNothingForAFailedClear(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{1: "de"}))

	_, err := recording.Clear(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), clicks.Blast{Cleared: []uint32{1, lastTile + 1}})
	require.ErrorIs(t, err, errOutOfRange)

	assert.Empty(t, written(events))
}

func TestRecordingNotesAClickThatTookATileAsATake(t *testing.T) {
	tiles := newTiles(map[uint32]string{1: "de"})
	recording, events := recorded(tiles)

	impact, err := recording.Click(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), 1, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Taken, impact.Outcome)
	assert.Equal(t, "fr", tiles.owners[1])
	assert.Equal(t, []ledger.Event{ledger.Taking{Tile: 1, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start}},
		written(events))
}

func TestRecordingNotesAClickOnAShieldAsAStrikeThatTakesNothing(t *testing.T) {
	tiles := newTiles(map[uint32]string{1: "de"})
	tiles.shields[1] = 2
	recording, events := recorded(tiles)

	impact, err := recording.Click(caller(t), 1, "fr")
	require.NoError(t, err)

	assert.Equal(t, clicks.Shielded, impact.Outcome)
	assert.Equal(t, []ledger.Event{ledger.Striking{
		Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Owner: "de", Shields: 1, At: start,
	}}, written(events))
	assert.Empty(t, replay(events), "the tile kept its flag")
}

func TestRecordingNotesTheCallersScopeAndOnlyAChange(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{1: "de", 2: "fr"}))

	ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::1")

	_, err := recording.Click(ctx, 1, "fr")
	require.NoError(t, err)
	_, err = recording.Click(ctx, 2, "fr")
	require.NoError(t, err)

	assert.Equal(t, []ledger.Event{ledger.Taking{Tile: 1, Scope: "2001:db8::/64", Country: "fr", Previous: "de", At: start}},
		written(events), "a v6 caller is its /64, and a tile it already held is no take")
}

func TestRecordingNotesNothingForACallerWithNoAddress(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{1: "de"}))

	_, err := recording.Click(t.Context(), 1, "fr")
	require.NoError(t, err)

	assert.Empty(t, written(events))
}

func TestRecordingKeepsEveryTake(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{7: "de"}))

	for _, click := range []struct{ address, flag string }{{"1.2.3.4", "fr"}, {"5.6.7.8", "de"}, {"1.2.3.4", "ps"}} {
		_, err := recording.Click(cpctx.AddIPToContext(t.Context(), click.address), 7, click.flag)
		require.NoError(t, err)
	}

	assert.Equal(t, []ledger.Taking{
		{Tile: 7, Scope: "1.2.3.4", Country: "fr", Previous: "de", At: start},
		{Tile: 7, Scope: "5.6.7.8", Country: "de", Previous: "fr", At: start},
		{Tile: 7, Scope: "1.2.3.4", Country: "ps", Previous: "de", At: start},
	}, replay(events))
}

func TestRecordingNotesNothingForAFailedWrite(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{}))

	_, err := recording.Click(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), lastTile+1, "fr")
	require.ErrorIs(t, err, errOutOfRange)

	assert.Empty(t, written(events))
}

func TestRecordingNotesTheAccountTheTokenNamed(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{1: "de"}))

	_, err := recording.Click(caller(t), 1, "fr")
	require.NoError(t, err)

	assert.Equal(t, []ledger.Taking{{Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Previous: "de", At: start}},
		replay(events))
}

func TestRecordingNotesASpreadAsOneEventWithWhatItTookAndWhatItStruck(t *testing.T) {
	tiles := newTiles(map[uint32]string{2: "de", 3: "pl", 4: "fr"})
	tiles.shields[3] = 1
	recording, events := recorded(tiles)

	require.NoError(t, recording.Spread(caller(t), 1, "fr", []uint32{2, 3, 4, 5}))

	assert.Equal(t, map[uint32]string{2: "fr", 3: "pl", 4: "fr", 5: "fr"}, tiles.owners)
	assert.Equal(t, []ledger.Event{ledger.Spreading{
		Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "de", Outcome: clicks.Taken},
			{Tile: 3, Owner: "pl", Outcome: clicks.Shielded},
			{Tile: 5, Outcome: clicks.Taken},
		},
	}}, written(events), "a tile the flag already held is left out")
	assert.Equal(t, []ledger.Taking{
		{Tile: 2, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Previous: "de", At: start},
		{Tile: 5, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", At: start},
	}, replay(events))
}

func TestRecordingNotesASpreadThatFailedWithWhatItTookBeforeIt(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{2: "de"}))

	err := recording.Spread(caller(t), 1, "fr", []uint32{2, lastTile + 1, 3})
	require.ErrorIs(t, err, errOutOfRange)

	assert.Equal(t, []ledger.Taking{
		{Tile: 2, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Previous: "de", At: start},
	}, replay(events))
}

func TestRecordingNotesASpreadOnALoneIslandThatTookNothing(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{}))

	require.NoError(t, recording.Spread(caller(t), 7, "fr", nil))

	assert.Equal(t, []ledger.Event{ledger.Spreading{Tile: 7, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", At: start}},
		written(events), "the spread click was spent")
}

func TestRecordingNotesAnEnclosureAsOneEventOnTheTileThatClosedIt(t *testing.T) {
	tiles := newTiles(map[uint32]string{8: "de", 9: "de"})
	tiles.shields[9] = 3
	recording, events := recorded(tiles)

	require.NoError(t, recording.Enclose(caller(t), 1, "fr", []uint32{8, 9}))

	assert.Equal(t, []ledger.Event{ledger.Enclosing{
		Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 8, Owner: "de", Outcome: clicks.Taken},
			{Tile: 9, Owner: "de", Outcome: clicks.Shielded, Shields: 2},
		},
	}}, written(events))
}

func TestRecordingNotesAShieldPlacedWithTheShieldsTheTileNowHolds(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{1: "fr"}))

	require.NoError(t, recording.Shield(caller(t), 1, "fr", 10))
	require.NoError(t, recording.Shield(caller(t), 1, "fr", 10))

	assert.Equal(t, []ledger.Event{
		ledger.Shielding{Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Shields: 1, At: start},
		ledger.Shielding{Tile: 1, Scope: "1.2.3.4", Account: "a-guest", Country: "fr", Shields: 2, At: start},
	}, written(events))
	assert.Empty(t, replay(events), "a shield takes no tile")
}

func TestRecordingNotesNothingForAShieldTheTileRefused(t *testing.T) {
	recording, events := recorded(newTiles(map[uint32]string{}))

	require.ErrorIs(t, recording.Shield(caller(t), lastTile+1, "fr", 10), errOutOfRange)

	assert.Empty(t, written(events))
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
	b.events = append(b.events,
		ledger.Taking{Tile: 7, Scope: "campus", Account: "guest", Country: "ps", Previous: "il"},
		ledger.Taking{Tile: 7, Scope: "home", Account: "guest", Country: "fr", Previous: "ps"},
		ledger.Taking{Tile: 8, Scope: "home", Account: "other", Country: "ps", Previous: "il"},
	)
	b.owners[7], b.owners[8] = "fr", "ps"

	runs := ledger.NewRuns(ledger.Caller{Account: "guest"})
	for _, event := range b.events {
		runs.See(event)
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

func TestAnAccountIDIsAUUIDThatIsNotNil(t *testing.T) {
	account, err := ledger.AccountIDOf("0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10")
	require.NoError(t, err)
	assert.Equal(t, "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", account.String())

	for _, value := range []string{"", "not-an-id", "00000000-0000-0000-0000-000000000000"} {
		_, err := ledger.AccountIDOf(value)
		assert.ErrorIs(t, err, ledger.ErrInvalidAccount, value)
	}
}
