package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

const vandal = "7a1d3c55-0e4b-4c8f-9d2a-6b5e4f3a2c10"

var painted = historySince.Add(time.Minute)

type seenThrough struct {
	takings []ledger.Taking
	strikes []ledger.Strike
	scenes  []ledger.Scene
}

func entryOn(t *testing.T, on clicks.Stage, event ledger.Event, forgotten bool) ledger.Entry {
	t.Helper()
	entry, err := event.Entry()
	require.NoError(t, err)
	entry.Stage, entry.Forgotten = on, forgotten
	return entry
}

func forgottenEntry(t *testing.T, event ledger.Event) ledger.Entry {
	t.Helper()
	return entryOn(t, clicks.WorldStage, event, true)
}

func keptEntry(t *testing.T, event ledger.Event) ledger.Entry {
	t.Helper()
	return entryOn(t, clicks.WorldStage, event, false)
}

func readThrough(t *testing.T, erasure *ledger.Erasure, entries ...ledger.Entry) seenThrough {
	t.Helper()
	var seen seenThrough
	for _, entry := range entries {
		event, kept, err := erasure.Read(entry)
		require.NoError(t, err)
		require.Equal(t, !entry.Forgotten, kept)
		if !kept {
			continue
		}
		event.Replay(func(taking ledger.Taking) { seen.takings = append(seen.takings, taking) })
		event.Strikes(func(strike ledger.Strike) { seen.strikes = append(seen.strikes, strike) })
		event.Show(func(scene ledger.Scene) { seen.scenes = append(seen.scenes, scene) })
	}
	return seen
}

func paint(tile uint32, previous, country string) ledger.Taking {
	return takeBy(vandal, tile, previous, country, painted)
}

func TestATakeOfAForgottenPaintTakesTheTileFromWhatItShowedBeforeAndATakeBackToItIsNone(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		forgottenEntry(t, paint(2, "ie", "cn")),
		keptEntry(t, takeBy(bob, 1, "cn", "fr", at)),
		keptEntry(t, takeBy(ada, 2, "cn", "ie", at)),
		keptEntry(t, takeBy(ada, 1, "fr", "ie", at.Add(time.Minute))),
	)

	assert.Equal(t, []ledger.Taking{takeBy(bob, 1, "ie", "fr", at), takeBy(ada, 1, "fr", "ie", at.Add(time.Minute))},
		seen.takings, "Ireland painting over the forgotten paint changed nothing; France's take came after it")
	require.Len(t, seen.scenes, 2)
	assert.Equal(t, "ie", seen.scenes[0].Change.Previous)
}

func TestAStrikeAShieldOrAFortifyOnAForgottenPaintIsNone(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		keptEntry(t, ledger.Striking{Tile: 1, Scope: "203.0.113.8", Account: bob, Country: "fr", Owner: "cn", At: at}),
		keptEntry(t, ledger.Shielding{Tile: 1, Scope: "203.0.113.9", Account: ada, Country: "cn", Shields: 1, At: at}),
		keptEntry(t, ledger.Fortifying{
			Tile: 2, Province: 4, Scope: "203.0.113.9", Account: ada, Country: "cn", At: at,
			Raised: []clicks.TileShields{{Tile: 1, Shields: 2}, {Tile: 2, Shields: 1}},
		}),
	)

	assert.Empty(t, seen.strikes)
	require.Len(t, seen.scenes, 1, "only tile 2 was China's beneath")
	assert.Equal(t, uint32(2), seen.scenes[0].Change.Tile)
}

func TestATileThatNoLongerWearsTheForgottenPaintIsReadAsItIs(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		forgottenEntry(t, paint(2, "ie", "cn")),
		keptEntry(t, takeBy(bob, 1, "ie", "fr", at)),
		keptEntry(t, takeBy(bob, 2, "", "fr", at)),
	)

	assert.Equal(t, []ledger.Taking{takeBy(bob, 1, "ie", "fr", at), takeBy(bob, 2, "", "fr", at)}, seen.takings,
		"a revert gave tile 1 back, a reset blanked tile 2: neither left an event")
}

func TestAForgottenPaintPaintedTwiceShowsWhatWasBeneathTheFirst(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		forgottenEntry(t, ledger.Bombing{Scope: "203.0.113.7", Account: vandal, At: painted, Blast: clicks.Blast{
			Tile: 1, CountryID: "cn", Cleared: []uint32{1}, Owners: []string{"cn"},
		}}),
		keptEntry(t, takeBy(bob, 1, "", "fr", at)),
	)

	assert.Equal(t, []ledger.Taking{takeBy(bob, 1, "ie", "fr", at)}, seen.takings)
}

func TestABombAndAnEnclosureSeeThroughTheForgottenPaint(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		forgottenEntry(t, paint(2, "ie", "cn")),
		forgottenEntry(t, paint(3, "", "cn")),
		forgottenEntry(t, paint(4, "ie", "cn")),
		forgottenEntry(t, paint(5, "de", "cn")),
		keptEntry(t, ledger.Bombing{Scope: "203.0.113.8", Account: bob, At: at, Blast: clicks.Blast{
			Tile: 1, CountryID: "fr", Cleared: []uint32{1, 3}, Owners: []string{"cn", "cn"},
			Struck: []uint32{2}, Left: []int{1}, StruckOwners: []string{"cn"},
		}}),
		keptEntry(t, ledger.Enclosing{Tile: 4, Scope: "203.0.113.9", Account: ada, Country: "ie", At: at, Impacts: []clicks.Impact{
			{Tile: 4, Owner: "cn", Outcome: clicks.Taken},
			{Tile: 5, Owner: "cn", Outcome: clicks.Taken},
			{Tile: 6, Owner: "fr", Outcome: clicks.Taken},
		}}),
	)

	require.Len(t, seen.scenes, 2)
	blast := seen.scenes[0].Blast
	assert.Equal(t, []uint32{1}, blast.Cleared, "tile 3 was nobody's beneath")
	assert.Equal(t, []string{"ie"}, blast.Owners)
	assert.Empty(t, blast.Struck, "the shield on tile 2 stood on the forgotten paint")
	assert.Equal(t, []clicks.Impact{
		{Tile: 5, Owner: "de", Outcome: clicks.Taken}, {Tile: 6, Owner: "fr", Outcome: clicks.Taken},
	}, seen.scenes[1].Enclosure.Impacts, "tile 4 was Ireland's beneath")
	assert.Equal(t, []ledger.Taking{
		{Tile: 1, Scope: "203.0.113.8", Account: bob, Previous: "ie", At: at},
		{Tile: 5, Scope: "203.0.113.9", Account: ada, Country: "ie", Previous: "de", At: at},
		{Tile: 6, Scope: "203.0.113.9", Account: ada, Country: "ie", Previous: "fr", At: at},
	}, seen.takings)
	assert.Empty(t, seen.strikes)
}

func TestAWipeOrAnotherStageShowsNoForgottenPaint(t *testing.T) {
	at := painted.Add(time.Minute)
	erasure := ledger.NewErasure()
	arena := clicks.MapID("a1").On(clicks.TestBoard)
	seen := readThrough(t, erasure,
		forgottenEntry(t, paint(1, "ie", "cn")),
		forgottenEntry(t, paint(2, "ie", "cn")),
		entryOn(t, arena, takeBy(bob, 1, "cn", "fr", at), false),
	)
	erasure.Wipe()
	seen.takings = append(seen.takings, readThrough(t, erasure, keptEntry(t, takeBy(bob, 2, "cn", "fr", at))).takings...)

	assert.Equal(t, []ledger.Taking{takeBy(bob, 1, "cn", "fr", at), takeBy(bob, 2, "cn", "fr", at)}, seen.takings)
}

func TestAnEventOverNoForgottenPaintIsReadAsItIs(t *testing.T) {
	at := painted.Add(time.Minute)
	seen := readThrough(t, ledger.NewErasure(),
		forgottenEntry(t, paint(1, "ie", "cn")),
		keptEntry(t, takeBy(bob, 2, "ie", "fr", at)),
		keptEntry(t, ledger.Striking{Tile: 3, Scope: "203.0.113.8", Account: bob, Country: "fr", Owner: "ie", At: at}),
	)

	assert.Equal(t, []ledger.Taking{takeBy(bob, 2, "ie", "fr", at)}, seen.takings)
	assert.Len(t, seen.strikes, 1)
	assert.Len(t, seen.scenes, 2)
}
