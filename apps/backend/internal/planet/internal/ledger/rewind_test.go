package ledger_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5c0ffee0-1d2e-4f3a-8b4c-5d6e7f8a9b0c"
)

func rewind(t *testing.T, world arenas.Played, events ...ledger.Event) *ledger.Rewind {
	t.Helper()
	rewound, err := ledger.NewRewind(historySince, historyUntil, time.Hour, world)
	require.NoError(t, err)
	for _, event := range events {
		entry, err := event.Entry()
		require.NoError(t, err)
		require.NoError(t, rewound.Record(entry))
	}
	return rewound
}

func takeBy(account string, tile uint32, previous, country string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: "203.0.113.7", Account: account, Country: country, Previous: previous, At: at}
}

func mapNow(owners ...string) clicks.DenseBatch {
	batch := clicks.DenseBatch{Start: 1, Codes: []string{""}}
	codes := map[string]uint16{"": 0}
	for _, owner := range owners {
		code, known := codes[owner]
		if !known {
			code = uint16(len(batch.Codes))
			codes[owner] = code
			batch.Codes = append(batch.Codes, owner)
		}
		batch.Tiles = binary.LittleEndian.AppendUint16(batch.Tiles, code)
	}
	return batch
}

func TestARewindOpensEachTileOnWhatItHeldBeforeItsFirstChangeAndEveryOtherOneAsItIsNow(t *testing.T) {
	cleared := historySince.Add(70 * time.Minute)
	rewound := rewind(t, arenas.Played{},
		takeBy(ada, 1, "fr", "de", historySince.Add(10*time.Minute)),
		takeBy(ada, 1, "de", "jp", historySince.Add(20*time.Minute)),
		ledger.Bombing{Scope: "203.0.113.8", Account: bob, At: cleared, Blast: clicks.Blast{
			Tile: 3, CountryID: "fr", Cleared: []uint32{3}, Owners: []string{"it"},
		}},
	)

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Owners: map[uint32]ledger.Owner{1: {Country: "fr"}, 2: {Country: "fr"}, 3: {Country: "it"}, 4: {Country: "jp"}}},
		{At: historySince.Add(time.Hour), Owners: map[uint32]ledger.Owner{
			1: {Country: "jp", Account: ada, At: historySince.Add(20 * time.Minute)},
		}},
		{At: historyUntil, Owners: map[uint32]ledger.Owner{3: {Account: bob, At: cleared}}},
	}, rewound.Frames(mapNow("jp", "fr", "", "jp")), "a bomb's clear is a change to no owner, by the bomber")
}

func TestARewindReadsTheTilesASpreadAnEncloseAndABombListAndNoOwnerFromAStrikeAShieldOrAFortify(t *testing.T) {
	at := historySince.Add(5 * time.Minute)
	rewound := rewind(t, arenas.Played{},
		takeBy(ada, 1, "fr", "de", at),
		ledger.Spreading{Tile: 1, Scope: "203.0.113.7", Account: ada, Country: "de", At: at, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "fr", Outcome: clicks.Taken}, {Tile: 3, Owner: "fr", Outcome: clicks.Shielded, Shields: 1},
		}},
		ledger.Enclosing{Tile: 4, Scope: "203.0.113.7", Account: ada, Country: "de", At: at, Impacts: []clicks.Impact{
			{Tile: 4, Owner: "", Outcome: clicks.Taken}, {Tile: 5, Owner: "fr", Outcome: clicks.Taken},
		}},
		ledger.Bombing{Scope: "203.0.113.8", Account: bob, At: at, Blast: clicks.Blast{
			Tile: 6, CountryID: "fr", Cleared: []uint32{6}, Owners: []string{"de"},
			Struck: []uint32{7}, Left: []int{2}, StruckOwners: []string{"de"},
		}},
		ledger.Striking{Tile: 3, Scope: "203.0.113.7", Account: ada, Country: "de", Owner: "fr", Shields: 0, At: at},
		ledger.Shielding{Tile: 7, Scope: "203.0.113.8", Account: bob, Country: "de", Shields: 3, At: at},
		ledger.Fortifying{
			Tile: 5, Landmass: 9, Ground: "fr", Tiles: 2, Scope: "203.0.113.7", Account: ada, Country: "de", At: at,
			Raised: []clicks.TileShields{{Tile: 4, Shields: 1}, {Tile: 5, Shields: 1}},
		},
	)

	frames := rewound.Frames(mapNow("de", "de", "fr", "de", "de", "", "de"))
	assert.Equal(t, map[uint32]ledger.Owner{
		1: {Country: "fr"}, 2: {Country: "fr"}, 3: {Country: "fr"}, 5: {Country: "fr"}, 6: {Country: "de"}, 7: {Country: "de"},
	}, frames[0].Owners, "tile 4 was nobody's before the enclose took it")
	assert.Equal(t, map[uint32]ledger.Owner{
		1: {Country: "de", Account: ada, At: at}, 2: {Country: "de", Account: ada, At: at},
		4: {Country: "de", Account: ada, At: at}, 5: {Country: "de", Account: ada, At: at},
		6: {Account: bob, At: at},
	}, frames[1].Owners)
	assert.Empty(t, frames[2].Owners)
	assert.Equal(t, map[uint32]uint32{1: 1, 2: 1, 3: 2, 4: 1, 5: 1, 6: 1, 7: 1}, rewound.Hits(),
		"tile 3 was struck by the spread and by a click, tile 7 by the bomb")
}

func TestATileChangedTwiceInAStepShowsItsLastChangeAndWhoMadeIt(t *testing.T) {
	rewound := rewind(t, arenas.Played{},
		takeBy(ada, 1, "fr", "de", historySince.Add(5*time.Minute)),
		takeBy(bob, 1, "de", "it", historySince.Add(20*time.Minute)),
	)

	assert.Equal(t, map[uint32]ledger.Owner{1: {Country: "it", Account: bob, At: historySince.Add(20 * time.Minute)}},
		rewound.Frames(mapNow("it"))[1].Owners)
}

func TestATileChangedBackWithinAStepIsInNoFrameAndAcrossTwoStepsIsInBoth(t *testing.T) {
	rewound := rewind(t, arenas.Played{},
		takeBy(ada, 1, "fr", "de", historySince.Add(5*time.Minute)),
		takeBy(bob, 1, "de", "fr", historySince.Add(20*time.Minute)),
		takeBy(ada, 2, "fr", "de", historySince.Add(50*time.Minute)),
		takeBy(bob, 2, "de", "fr", historySince.Add(70*time.Minute)),
	)

	frames := rewound.Frames(mapNow("fr", "fr"))
	assert.Equal(t, map[uint32]ledger.Owner{1: {Country: "fr"}, 2: {Country: "fr"}}, frames[0].Owners)
	assert.Equal(t, map[uint32]ledger.Owner{2: {Country: "de", Account: ada, At: historySince.Add(50 * time.Minute)}}, frames[1].Owners)
	assert.Equal(t, map[uint32]ledger.Owner{2: {Country: "fr", Account: bob, At: historySince.Add(70 * time.Minute)}}, frames[2].Owners)
}

func TestAnEntryAtAStepsTimeIsInTheNextFrameAndOneFromTheEndOnOnlyRewinds(t *testing.T) {
	rewound := rewind(t, arenas.Played{},
		takeBy(ada, 1, "fr", "de", historySince.Add(time.Hour)),
		takeBy(ada, 2, "fr", "de", historyUntil),
		takeBy(ada, 1, "de", "jp", historyUntil.Add(time.Hour)),
	)

	frames := rewound.Frames(mapNow("jp", "de"))
	assert.Equal(t, map[uint32]ledger.Owner{1: {Country: "fr"}, 2: {Country: "fr"}}, frames[0].Owners)
	assert.Empty(t, frames[1].Owners)
	assert.Equal(t, map[uint32]ledger.Owner{1: {Country: "de", Account: ada, At: historySince.Add(time.Hour)}}, frames[2].Owners)
	assert.Equal(t, map[uint32]uint32{1: 1}, rewound.Hits())
}

func TestARewindLeavesOutWhatWasPlayedOnAnArenasMap(t *testing.T) {
	opened, closed := historySince.Add(10*time.Minute), historySince.Add(40*time.Minute)
	world := arenas.MapsOf([]arenas.Stretch{{ID: "a1", OpenedAt: opened, ClosedAt: closed, Closed: true}}).World()
	rewound := rewind(t, world,
		takeBy(ada, 1, "", "de", historySince.Add(20*time.Minute)),
		takeBy(ada, 2, "fr", "de", historySince.Add(50*time.Minute)),
	)

	frames := rewound.Frames(mapNow("fr", "de"))
	assert.Equal(t, map[uint32]ledger.Owner{1: {Country: "fr"}, 2: {Country: "fr"}}, frames[0].Owners)
	assert.Equal(t, map[uint32]ledger.Owner{2: {Country: "de", Account: ada, At: historySince.Add(50 * time.Minute)}}, frames[1].Owners)
	assert.Equal(t, map[uint32]uint32{2: 1}, rewound.Hits())
}

func TestARewindOfNoTimeIsTheMapAlone(t *testing.T) {
	rewound, err := ledger.NewRewind(historySince, historySince, time.Hour, arenas.Played{})
	require.NoError(t, err)
	entry, err := takeBy(ada, 1, "fr", "de", historySince).Entry()
	require.NoError(t, err)
	require.NoError(t, rewound.Record(entry))

	assert.Equal(t, []ledger.Frame{{At: historySince, Owners: map[uint32]ledger.Owner{1: {Country: "fr"}}}},
		rewound.Frames(mapNow("de")))
}

func TestARewindStartsNoLaterThanItEndsAndMovesByAStep(t *testing.T) {
	_, err := ledger.NewRewind(historyUntil, historySince, time.Hour, arenas.Played{})
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
	_, err = ledger.NewRewind(historySince, historyUntil, 0, arenas.Played{})
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
}
