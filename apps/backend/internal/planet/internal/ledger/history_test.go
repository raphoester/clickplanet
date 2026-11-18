package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var (
	historySince = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	historyUntil = historySince.Add(90 * time.Minute)
)

func record(t *testing.T, history *ledger.History, events ...ledger.Event) {
	t.Helper()
	recordOn(t, history, clicks.WorldStage, events...)
}

// recordOn records events made on a stage: their map, on the board it played.
func recordOn(t *testing.T, history *ledger.History, on clicks.Stage, events ...ledger.Event) {
	t.Helper()
	for _, event := range events {
		entry, err := event.Entry()
		require.NoError(t, err)
		entry.Stage = on
		require.False(t, history.Done(entry))
		require.NoError(t, history.Record(entry))
	}
}

func take(tile uint32, country string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: "203.0.113.7", Country: country, At: at}
}

func TestAHistoryOpensOnEveryTileHeldThenKeepsWhatChangedAtEachStep(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)

	record(t, history,
		take(1, "fr", historySince.Add(-48*time.Hour)),
		take(2, "de", historySince.Add(-time.Hour)),
		take(1, "de", historySince.Add(10*time.Minute)),
		take(1, "fr", historySince.Add(20*time.Minute)),
		take(3, "jp", historySince.Add(time.Hour)),
		ledger.Bombing{Scope: "203.0.113.7", At: historySince.Add(80 * time.Minute), Blast: clicks.Blast{
			Tile: 2, CountryID: "fr", Cleared: []uint32{2}, Owners: []string{"de"},
		}},
	)

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Whole: true, Owners: map[uint32]ledger.Owner{
			1: {Country: "fr", At: historySince.Add(-48 * time.Hour)}, 2: {Country: "de", At: historySince.Add(-time.Hour)},
		}},
		{At: historySince.Add(time.Hour), Owners: map[uint32]ledger.Owner{}},
		{At: historyUntil, Owners: map[uint32]ledger.Owner{
			2: {At: historySince.Add(80 * time.Minute)}, 3: {Country: "jp", At: historySince.Add(time.Hour)},
		}},
	}, history.Frames(), "tile 1 went back to France within the hour; a take at a step's time is in the next frame")
}

func TestATileKeepsWhenItWentToItsOwnerLastSoAnEnclosuresTilesLandTogether(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)

	enclosed := historySince.Add(70 * time.Minute)
	record(t, history,
		take(1, "de", historySince.Add(65*time.Minute)),
		take(1, "cz", historySince.Add(66*time.Minute)),
		take(1, "de", historySince.Add(68*time.Minute)),
		ledger.Enclosing{Tile: 3, Scope: "203.0.113.7", Country: "de", At: enclosed, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "cz", Outcome: clicks.Taken}, {Tile: 3, Owner: "cz", Outcome: clicks.Taken},
		}},
	)

	assert.Equal(t, map[uint32]ledger.Owner{
		1: {Country: "de", At: historySince.Add(68 * time.Minute)}, 2: {Country: "de", At: enclosed}, 3: {Country: "de", At: enclosed},
	}, history.Frames()[2].Owners)
}

func TestAHistoryCountsEachTilesChangesAndShieldHitsDuringTheWindowOnly(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)

	record(t, history,
		take(1, "fr", historySince.Add(-time.Minute)),
		take(1, "de", historySince.Add(time.Minute)),
		take(1, "fr", historySince.Add(2*time.Minute)),
		ledger.Striking{Tile: 4, Scope: "203.0.113.7", Country: "fr", Owner: "de", Shields: 1, At: historySince.Add(3 * time.Minute)},
	)

	assert.Equal(t, map[ledger.EraTile]uint32{{Tile: 1}: 2, {Tile: 4}: 1}, history.Hits())
}

func TestAWipeAtTheWindowsStartOpensItOnAnEmptyMap(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)

	record(t, history, take(1, "fr", historySince.Add(-time.Hour)))
	history.Wipe(historySince)
	record(t, history, take(2, "de", historySince.Add(time.Minute)))

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Whole: true, Owners: map[uint32]ledger.Owner{}},
		{At: historySince.Add(time.Hour), Owners: map[uint32]ledger.Owner{2: {Country: "de", At: historySince.Add(time.Minute)}}},
		{At: historyUntil, Owners: map[uint32]ledger.Owner{}},
	}, history.Frames())
}

func TestAHistoryIsDoneWithTheEntriesFromItsEnd(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)

	entry, err := take(1, "fr", historyUntil).Entry()
	require.NoError(t, err)
	assert.True(t, history.Done(entry))
}

func TestAHistoryStartsBeforeItEndsAndMovesByAStep(t *testing.T) {
	_, err := ledger.NewHistory(historyUntil, historySince, time.Hour, clicks.WorldStage)
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
	_, err = ledger.NewHistory(historySince, historyUntil, 0, clicks.WorldStage)
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
}

func TestAnArenasTakesChangeNoFrameNorHitAndTheWorldJoinsUpAroundIt(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, 30*time.Minute, clicks.WorldStage)
	require.NoError(t, err)
	battle := clicks.MapID("battle").On(clicks.TestBoard)

	record(t, history,
		take(1, "fr", historySince.Add(-time.Hour)),
		take(2, "de", historySince.Add(10*time.Minute)),
	)
	recordOn(t, history, battle,
		take(1, "br", historySince.Add(25*time.Minute)),
		take(3, "br", historySince.Add(40*time.Minute)),
		ledger.KnockingOut{
			Tile: 3, Scope: "203.0.113.7", Country: "ar", Out: "br", At: historySince.Add(45 * time.Minute),
			Cleared: []uint32{4},
		},
	)
	record(t, history, take(2, "it", historySince.Add(46*time.Minute)))
	recordOn(t, history, battle, take(5, "ar", historySince.Add(47*time.Minute)))

	frames := history.Frames()
	assert.Equal(t, map[uint32]ledger.Owner{2: {Country: "de", At: historySince.Add(10 * time.Minute)}}, frames[1].Owners,
		"the battle's first take changed nothing on the world's map")
	assert.Equal(t, map[uint32]ledger.Owner{2: {Country: "it", At: historySince.Add(46 * time.Minute)}}, frames[2].Owners,
		"the battle's takes and its knock-out changed nothing, and a world take among them is the world's: each names its map")
	assert.Equal(t, map[ledger.EraTile]uint32{{Tile: 2}: 2}, history.Hits(), "no hit where only the arena fought")
}

func TestAHistoryAcrossTwoErasDrawsEachOnItsBoardWithTheCut(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, 30*time.Minute, clicks.WorldStage)
	require.NoError(t, err)
	fixture := clicks.World.On(clicks.TestBoard + 2)
	cut := historySince.Add(40 * time.Minute)

	record(t, history,
		take(1, "fr", historySince.Add(-time.Hour)),
		take(2, "de", historySince.Add(10*time.Minute)),
	)
	recordOn(t, history, fixture, take(7, "jp", historySince.Add(20*time.Minute)))
	history.Cut(cut, fixture.Board, carried{})
	recordOn(t, history, fixture,
		take(3, "it", historySince.Add(50*time.Minute)),
		take(3, "es", historySince.Add(55*time.Minute)),
	)
	record(t, history, take(2, "pt", historySince.Add(60*time.Minute)))

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Whole: true, Owners: map[uint32]ledger.Owner{1: {Country: "fr", At: historySince.Add(-time.Hour)}}},
		{At: historySince.Add(30 * time.Minute), Owners: map[uint32]ledger.Owner{
			2: {Country: "de", At: historySince.Add(10 * time.Minute)},
		}},
		{At: historySince.Add(60 * time.Minute), Whole: true, Owners: map[uint32]ledger.Owner{
			3: {Country: "es", At: historySince.Add(55 * time.Minute)},
		}},
		{At: historySince.Add(90 * time.Minute), Owners: map[uint32]ledger.Owner{}},
	}, history.Frames(), "the frames before the cut are earth-1's, untouched by the fixture's take before its era; "+
		"the first after it holds every tile held on the fixture, whole, and an earth-1 take after the cut is no longer the map's")
	assert.Equal(t, map[ledger.EraTile]uint32{{Era: 0, Tile: 2}: 1, {Era: 1, Tile: 3}: 2}, history.Hits(),
		"each era's hits on its own board")
}

func TestTheFirstFrameAfterACutHoldsEveryTileHeldOnTheNewBoard(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)
	fixture := clicks.World.On(clicks.TestBoard + 2)

	record(t, history, take(1, "fr", historySince.Add(10*time.Minute)))
	history.Cut(historySince.Add(20*time.Minute), fixture.Board, carried{})
	recordOn(t, history, fixture,
		take(5, "de", historySince.Add(30*time.Minute)),
		take(6, "de", historySince.Add(31*time.Minute)),
	)
	recordOn(t, history, fixture, take(5, "de", historySince.Add(70*time.Minute)))

	frames := history.Frames()
	require.Len(t, frames, 3)
	assert.Equal(t, map[uint32]ledger.Owner{
		5: {Country: "de", At: historySince.Add(30 * time.Minute)}, 6: {Country: "de", At: historySince.Add(31 * time.Minute)},
	}, frames[1].Owners, "whole: tile 1 of earth-1 is gone, and no frame says so tile by tile")
	assert.True(t, frames[1].Whole)
	assert.Equal(t, map[uint32]ledger.Owner{}, frames[2].Owners, "a take that changes no owner after it changes no frame")
}

func TestACutAtOrBeforeTheWindowsStartIsTheEraInForceAtIt(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)
	fixture := clicks.World.On(clicks.TestBoard + 2)

	record(t, history, take(1, "fr", historySince.Add(-2*time.Hour)))
	history.Cut(historySince.Add(-time.Hour), fixture.Board, carried{})
	recordOn(t, history, fixture, take(4, "de", historySince.Add(-time.Minute)), take(4, "it", historySince.Add(time.Minute)))

	assert.Equal(t, map[uint32]ledger.Owner{4: {Country: "de", At: historySince.Add(-time.Minute)}}, history.Frames()[0].Owners,
		"the window opens on the fixture's map")
	assert.Equal(t, map[ledger.EraTile]uint32{{Era: 0, Tile: 4}: 1}, history.Hits(), "its first era")
}

func TestACutCarriesEachTileOverFromTheTileItComesFrom(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour, clicks.WorldStage)
	require.NoError(t, err)
	fixture := clicks.World.On(clicks.TestBoard + 2)
	switched := historySince.Add(20 * time.Minute)

	record(t, history,
		take(1, "fr", historySince.Add(-time.Hour)),
		take(2, "de", historySince.Add(10*time.Minute)),
	)
	history.Cut(switched, fixture.Board, carried{10: 1, 11: 1, 12: 2, 13: 3})
	recordOn(t, history, fixture, take(11, "it", historySince.Add(30*time.Minute)))

	frames := history.Frames()
	assert.Equal(t, ledger.Frame{At: historySince.Add(time.Hour), Whole: true, Owners: map[uint32]ledger.Owner{
		10: {Country: "fr", At: switched}, 11: {Country: "it", At: historySince.Add(30 * time.Minute)},
		12: {Country: "de", At: switched},
	}}, frames[1], "each child of a held tile holds its flag, a child of a blank tile and a tile with no parent start blank")
	assert.Equal(t, map[ledger.EraTile]uint32{{Era: 0, Tile: 2}: 1, {Era: 1, Tile: 11}: 1}, history.Hits(),
		"the carry-over is no take: it hits nothing")
}

// carried names each tile of the board after a switch and the tile it comes from.
type carried map[uint32]uint32

func (c carried) Each(visit func(child, parent uint32)) {
	for child, parent := range c {
		visit(child, parent)
	}
}
