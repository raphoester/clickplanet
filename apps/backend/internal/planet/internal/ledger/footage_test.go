package ledger_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func denseOf(owners ...string) clicks.DenseBatch {
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

func ownersOf(batch clicks.DenseBatch) []string {
	owners := make([]string, len(batch.Tiles)/2)
	for i := range owners {
		owners[i] = batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[i*2:])]
	}
	return owners
}

func footageOf(since, until time.Time, events ...ledger.Event) *ledger.Footage {
	footage := ledger.NewFootage(since, until)
	for _, event := range events {
		footage.See(event)
	}
	return footage
}

func TestTheOpeningIsWhatEachTileHeldBeforeItsFirstChangeAfterSince(t *testing.T) {
	since := start.Add(time.Minute)
	footage := footageOf(since, start.Add(time.Hour),
		ledger.Taking{Tile: 1, Country: "it", At: start},
		ledger.Taking{Tile: 1, Country: "fr", Previous: "it", At: since},
		ledger.Taking{Tile: 1, Country: "de", Previous: "fr", At: since.Add(time.Second)},
		ledger.Taking{Tile: 2, Country: "fr", At: since.Add(time.Second)},
	)

	opening := footage.Opening(denseOf("de", "fr", "pl"))

	assert.Equal(t, []string{"it", "", "pl"}, ownersOf(opening))
}

func TestTheOpeningRewindsPastTheEndOfTheWindow(t *testing.T) {
	until := start.Add(time.Minute)
	footage := footageOf(start, until,
		ledger.Taking{Tile: 1, Country: "fr", Previous: "it", At: start},
		ledger.Taking{Tile: 1, Country: "de", Previous: "fr", At: until.Add(time.Hour)},
	)

	assert.Equal(t, []string{"it"}, ownersOf(footage.Opening(denseOf("de"))))
	assert.Len(t, footage.Scenes(), 1, "only what happened inside the window is shown")
}

func TestABombsClearedTilesOpenOnTheirOwnersAndItsStruckTilesWithTheShieldItTook(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour), ledger.Bombing{At: start, Blast: clicks.Blast{
		Tile: 1, CountryID: "fr", Cleared: []uint32{1, 2}, Owners: []string{"de", "pl"},
		Struck: []uint32{3}, Left: []int{2},
	}})

	now := denseOf("", "", "it")
	now.Shields = []clicks.TileShields{{Tile: 3, Shields: 2}}
	opening := footage.Opening(now)

	assert.Equal(t, []string{"de", "pl", "it"}, ownersOf(opening))
	assert.Equal(t, []clicks.TileShields{{Tile: 3, Shields: 3}}, opening.Shields)
}

func TestShieldsOpenAsTheyStoodBeforeTheirFirstChange(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour),
		ledger.Shielding{Tile: 1, Country: "fr", Shields: 1, At: start},
		ledger.Striking{Tile: 2, Country: "de", Owner: "fr", Shields: 4, At: start},
		ledger.Striking{Tile: 2, Country: "de", Owner: "fr", Shields: 3, At: start.Add(time.Second)},
	)

	now := denseOf("fr", "fr", "fr")
	now.Shields = []clicks.TileShields{{Tile: 1, Shields: 1}, {Tile: 2, Shields: 3}, {Tile: 3, Shields: 7}}
	opening := footage.Opening(now)

	assert.Equal(t, []clicks.TileShields{{Tile: 2, Shields: 5}, {Tile: 3, Shields: 7}}, opening.Shields)
	assert.Equal(t, []string{"fr", "fr", "fr"}, ownersOf(opening))
}

func TestAnOwnerTheMapNoLongerHoldsIsAddedToTheCodes(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour), ledger.Taking{Tile: 1, Country: "fr", Previous: "ps", At: start})

	opening := footage.Opening(denseOf("fr"))

	assert.Equal(t, []string{"ps"}, ownersOf(opening))
	assert.Equal(t, []string{"", "fr"}, denseOf("fr").Codes, "the map it was given is left as it was")
}

func TestAClickWithASpreadShowsEachTileItTookThenTheSpread(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour), ledger.Spreading{
		Tile: 1, Country: "fr", At: start, Impacts: []clicks.Impact{
			{Tile: 2, Owner: "de", Outcome: clicks.Taken},
			{Tile: 3, Owner: "pl", Outcome: clicks.Shielded, Shields: 1},
		},
	})

	assert.Equal(t, []ledger.Scene{
		{At: start, Change: &ledger.Change{TileUpdate: clicks.TileUpdate{Tile: 2, Value: "fr", Previous: "de"}}},
		{At: start, Change: &ledger.Change{
			TileUpdate: clicks.TileUpdate{Tile: 3, Value: "pl", Previous: "pl", Shields: 1}, Was: 2,
		}},
		{At: start, Spread: &ledger.Bonus{Tile: 1, Country: "fr", Tiles: []uint32{2, 3}}},
	}, footage.Scenes())
}

func TestAClickShowsAsAClickedTile(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour),
		ledger.Taking{Tile: 1, Country: "fr", Previous: "de", At: start},
		ledger.Striking{Tile: 2, Country: "fr", Owner: "de", Shields: 0, At: start},
	)

	assert.Equal(t, []ledger.Scene{
		{At: start, Change: &ledger.Change{TileUpdate: clicks.TileUpdate{Tile: 1, Value: "fr", Previous: "de", Clicked: true}}},
		{At: start, Change: &ledger.Change{
			TileUpdate: clicks.TileUpdate{Tile: 2, Value: "de", Previous: "de", Clicked: true}, Was: 1,
		}},
	}, footage.Scenes())
}

func TestAnEnclosureShowsAsItsTilesThenTheEnclosure(t *testing.T) {
	footage := footageOf(start, start.Add(time.Hour), ledger.Enclosing{
		Tile: 9, Country: "fr", At: start, Impacts: []clicks.Impact{{Tile: 4, Outcome: clicks.Taken}},
	})

	assert.Equal(t, []ledger.Scene{
		{At: start, Change: &ledger.Change{TileUpdate: clicks.TileUpdate{Tile: 4, Value: "fr"}}},
		{At: start, Enclosure: &ledger.Bonus{Tile: 9, Country: "fr", Tiles: []uint32{4}}},
	}, footage.Scenes())
}
