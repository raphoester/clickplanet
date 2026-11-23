package ledger_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type board struct {
	t       *testing.T
	live    owners
	entries []ledger.Entry
}

func newBoard(t *testing.T, initial owners) *board {
	t.Helper()
	b := &board{t: t, live: owners{}}
	for _, tile := range slices.Sorted(maps.Keys(initial)) {
		b.record(ledger.Taking{Tile: tile, Scope: "seed", Country: initial[tile]})
		b.live[tile] = initial[tile]
	}
	return b
}

func (b *board) record(event ledger.Event) {
	entry, err := event.Entry()
	require.NoError(b.t, err)
	entry.Stage = ledger.StageOf(event)
	b.entries = append(b.entries, entry)
}

func (b *board) take(tile uint32, scope, country string) {
	b.record(ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: b.live[tile]})
	b.live[tile] = country
}

func (b *board) bomb(scope string, tiles ...uint32) {
	blast := clicks.Blast{Cleared: tiles}
	for _, tile := range tiles {
		blast.Owners = append(blast.Owners, b.live[tile])
		b.live[tile] = ""
	}
	b.record(ledger.Bombing{Scope: scope, Blast: blast})
}

func (b *board) forget(caller ledger.Caller) {
	for i, entry := range b.entries {
		if caller.Made(ledger.Taking{Scope: entry.Scope, Account: entry.Account}) {
			b.entries[i].Forgotten = true
		}
	}
}

func (b *board) undo(caller ledger.Caller) *ledger.Undoing {
	undoing := ledger.NewUndoing(caller, clicks.WorldStage)
	for _, entry := range b.entries {
		require.NoError(b.t, undoing.Record(entry))
	}
	return undoing
}

func (b *board) restorations(scope string) []clicks.Restoration {
	return b.undo(ledger.Caller{Scope: scope}).Restorations(b.live, clicks.WorldStage)
}

func TestTwoScopesPaintingTheSameFlagDoNotHoldEachOthersTiles(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "B", "il")
	b.take(7, "C", "ps")

	assert.Empty(t, b.restorations("A"), "the tile wears A's flag, but C painted it")
	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}}, b.restorations("C"))
}

func TestARevertOfABomberGoesBackPastItsOwnTakes(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	b.bomb("A", 7)

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "", To: "il"}}, b.restorations("A"))
}

func TestABombOfAnotherScopeEndsWhatTheRevertGivesBack(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	b.bomb("B", 7)
	b.take(7, "A", "ps")

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: ""}}, b.restorations("A"))
}

func TestARevertOfTakesThatEndWhereTheyStartedGivesNothingBack(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "A", "il")

	assert.Empty(t, b.restorations("A"))
	assert.Zero(t, b.undo(ledger.Caller{Scope: "A"}).Touched())
}

func TestARevertOfAnAccountFollowsItAcrossScopes(t *testing.T) {
	b := newBoard(t, owners{7: "il", 8: "il"})
	b.record(ledger.Taking{Tile: 7, Scope: "campus", Account: "guest", Country: "ps", Previous: "il"})
	b.record(ledger.Taking{Tile: 7, Scope: "home", Account: "guest", Country: "fr", Previous: "ps"})
	b.record(ledger.Taking{Tile: 8, Scope: "home", Account: "other", Country: "ps", Previous: "il"})
	b.live[7], b.live[8] = "fr", "ps"

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "fr", To: "il"}},
		b.undo(ledger.Caller{Account: "guest"}).Restorations(b.live, clicks.WorldStage))
}

func TestAnArenaTakeNeitherGivesBackNorKeepsATileOfTheWorldsMap(t *testing.T) {
	battle := clicks.MapID("battle").On(clicks.TestBoard)
	b := newBoard(t, owners{7: "il", 8: "ps"})
	b.take(7, "A", "ps")
	b.record(ledger.Staged{On: battle, Event: ledger.Taking{Tile: 7, Scope: "B", Country: "de", At: start.Add(11 * time.Minute)}})
	b.record(ledger.Staged{On: battle, Event: ledger.Taking{Tile: 8, Scope: "A", Country: "ps", At: start.Add(12 * time.Minute)}})
	b.record(ledger.Staged{
		On:    clicks.World.On(clicks.TestBoard + 1),
		Event: ledger.Taking{Tile: 8, Scope: "A", Country: "ps", At: start.Add(13 * time.Minute)},
	})

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "ps", To: "il"}}, b.restorations("A"),
		"tile 8 is the world's Palestine: the arena take of it, and one on another board, is none of the world's")
}

func TestARevertRunAgainGivesBackWhatTheOneBeforeDidNotReach(t *testing.T) {
	b := newBoard(t, owners{7: "se", 8: "se"})
	b.take(7, "A", "de")
	b.take(8, "A", "de")
	b.forget(ledger.Caller{Scope: "A"})
	b.live[8] = "se"

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "de", To: "se"}}, b.restorations("A"),
		"the revert before gave tile 8 back and forgot both takes; tile 7 was too old for it")
}

func TestARevertSeesThroughThePaintOfAPlayerRevertedBefore(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	b.take(7, "B", "de")
	b.forget(ledger.Caller{Scope: "A"})

	assert.Equal(t, []clicks.Restoration{{Tile: 7, From: "de", To: "il"}}, b.restorations("B"),
		"what was under A's forgotten paint, never the paint")
}

type carry map[uint32]uint32

func (c carry) Each(visit func(child, parent uint32)) {
	for _, child := range slices.Sorted(maps.Keys(c)) {
		visit(child, c[child])
	}
}

func TestARevertGivesBackOnTheNewBoardATakeTheSwitchCarriedOver(t *testing.T) {
	earth2 := clicks.TestBoard + 1
	b := newBoard(t, owners{1: "lv", 2: "lv"})
	b.take(1, "A", "ie")
	undoing := b.undo(ledger.Caller{Scope: "A"})
	undoing.Cut(start, earth2, carry{10: 1, 11: 1, 12: 2})
	stage := clicks.World.On(earth2)
	entry, err := ledger.Taking{Tile: 11, Scope: "B", Country: "fr", Previous: "ie"}.Entry()
	require.NoError(t, err)
	entry.Stage = stage
	require.NoError(t, undoing.Record(entry))

	assert.Equal(t, []clicks.Restoration{{Tile: 10, From: "ie", To: "lv"}},
		undoing.Restorations(owners{10: "ie", 11: "fr", 12: "lv"}, stage),
		"tile 10 still wears A's take of earth-1's tile 1; B took tile 11 since")
	assert.Empty(t, undoing.Restorations(owners{10: "ie"}, clicks.WorldStage), "earth-1 plays no more")
}

func TestARevertGivesBackNothingAResetBlanked(t *testing.T) {
	b := newBoard(t, owners{7: "il"})
	b.take(7, "A", "ps")
	undoing := b.undo(ledger.Caller{Scope: "A"})
	undoing.Wipe(start)

	assert.Empty(t, undoing.Restorations(owners{}, clicks.WorldStage))
	assert.Zero(t, undoing.Touched())
}
