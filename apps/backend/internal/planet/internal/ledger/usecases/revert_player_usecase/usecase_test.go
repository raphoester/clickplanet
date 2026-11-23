package revert_player_usecase_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
)

type stubMap struct {
	// The world's on earth-1 when unset.
	playing clicks.Stage
	owners  map[uint32]string
	batches []int
	err     error
}

func (m *stubMap) Stage() clicks.Stage {
	if m.playing == (clicks.Stage{}) {
		return clicks.WorldStage
	}
	return m.playing
}

func (m *stubMap) Owner(tile uint32) (string, bool) { return m.owners[tile], true }

func (m *stubMap) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	m.batches = append(m.batches, len(restorations))
	if m.err != nil {
		return 0, m.err
	}

	restored := 0
	for _, r := range restorations {
		if m.owners[r.Tile] == r.From {
			m.owners[r.Tile] = r.To
			restored++
		}
	}
	return restored, nil
}

// world is the ledger postgres keeps from the blank map, with the switches of board in it, and the marks a revert
// sets on it.
type world struct {
	t        *testing.T
	entries  []ledger.Entry
	switches map[int]func(ledger.Replayed)
}

func newWorld(t *testing.T) *world {
	return &world{t: t, switches: map[int]func(ledger.Replayed){}}
}

func (w *world) Replay(_ context.Context, start func(first clicks.Stage) ledger.Replayed) (ledger.Position, error) {
	replayed := start(clicks.WorldStage)
	for i, entry := range w.entries {
		if change, ok := w.switches[i]; ok {
			change(replayed)
		}
		if err := replayed.Record(entry); err != nil {
			return 0, fmt.Errorf("failed to replay an event: %w", err)
		}
	}
	if change, ok := w.switches[len(w.entries)]; ok {
		change(replayed)
	}
	return ledger.Position(len(w.entries)), nil
}

func (w *world) Forget(caller ledger.Caller, before ledger.Position) {
	for i := range w.entries[:before] {
		if caller.Made(ledger.Taking{Scope: w.entries[i].Scope, Account: w.entries[i].Account}) {
			w.entries[i].Forgotten = true
		}
	}
}

func (w *world) record(on clicks.Stage, event ledger.Event) {
	entry, err := event.Entry()
	require.NoError(w.t, err)
	entry.Stage = on
	w.entries = append(w.entries, entry)
}

// seed takes each tile from blank, as a season's first takes do.
func (w *world) seed(tiles *stubMap) {
	for _, tile := range slices.Sorted(maps.Keys(tiles.owners)) {
		if country := tiles.owners[tile]; country != "" {
			w.record(clicks.WorldStage, ledger.Taking{Tile: tile, Scope: "seed", Country: country})
		}
	}
}

func (w *world) take(tiles *stubMap, tile uint32, scope, country string) {
	w.record(tiles.Stage(), ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: tiles.owners[tile]})
	tiles.owners[tile] = country
}

func (w *world) kept(scope string) int {
	n := 0
	for _, entry := range w.entries {
		if entry.Scope == scope && !entry.Forgotten {
			n++
		}
	}
	return n
}

func setup(t *testing.T) (*world, *stubMap) {
	t.Helper()

	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: "il"}}
	book.seed(tiles)
	for tile := uint32(1); tile <= 5; tile++ {
		book.take(tiles, tile, "9.9.9.9", "ps")
	}
	book.take(tiles, 4, "1.1.1.1", "il")
	tiles.owners[5] = ""

	return book, tiles
}

func revert(book *world, tiles *stubMap, batch int) *revert_player_usecase.UseCase {
	return revert_player_usecase.New(book, book, tiles, clicks.Pacing{Batch: batch})
}

func TestItGivesBackOnlyTheTilesStillWearingThePaintInBatches(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert(book, tiles, 2)

	out, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 4, Held: 3, Restored: 3}, out,
		"tile 4 is Israel's with or without the scope")
	assert.Equal(t, map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: ""}, tiles.owners)
	assert.Equal(t, []int{2, 1}, tiles.batches)
	assert.Zero(t, book.kept("9.9.9.9"), "a reverted scope's events are forgotten")
	assert.Equal(t, 1, book.kept("1.1.1.1"))

	again, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)
	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 4}, again, "the map holds none of its paint")
}

func TestARevertOfAPlayerWhoPaintedOverARevertedOneGivesBackWhatWasBeneathThePaint(t *testing.T) {
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "il"}}
	book.seed(tiles)
	book.take(tiles, 1, "9.9.9.9", "ps")
	book.take(tiles, 1, "1.1.1.1", "de")

	_, err := revert(book, tiles, 2).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)
	out, err := revert(book, tiles, 2).Execute(t.Context(), revert_player_usecase.In{Scope: "1.1.1.1"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Restored)
	assert.Equal(t, map[uint32]string{1: "il"}, tiles.owners, "the forgotten paint never comes back")
}

func TestADryRunCountsAndRestoresNothing(t *testing.T) {
	book, tiles := setup(t)

	out, err := revert(book, tiles, 2).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9", DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 4, Held: 3}, out)
	assert.Empty(t, tiles.batches)
	assert.Equal(t, 5, book.kept("9.9.9.9"))
}

func TestItGivesBackOnlyTheScopesLatestRunWhenTakesInterleave(t *testing.T) {
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{7: "il", 8: "il"}}
	book.seed(tiles)
	book.take(tiles, 7, "9.9.9.9", "ps")
	book.take(tiles, 7, "1.1.1.1", "de")
	book.take(tiles, 7, "9.9.9.9", "ps")
	book.take(tiles, 7, "9.9.9.9", "fr")
	book.take(tiles, 8, "9.9.9.9", "ps")
	book.take(tiles, 8, "1.1.1.1", "il")

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 1, Held: 1, Restored: 1}, out)
	assert.Equal(t, map[uint32]string{7: "de", 8: "il"}, tiles.owners,
		"tile 7 goes back to the retake that broke the run, and tile 8 was taken back already")
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	book, tiles := setup(t)

	_, err := revert(book, tiles, 2).Execute(t.Context(), revert_player_usecase.In{Scope: "bot"})
	require.ErrorIs(t, err, ledger.ErrInvalidScope)
}

func TestItStopsWhenTheContextEndsAndSaysHowFarItGot(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player_usecase.New(book, book, tiles, clicks.Pacing{Batch: 2, Pause: time.Hour})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := useCase.Execute(ctx, revert_player_usecase.In{Scope: "9.9.9.9"})
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, 2, out.Restored, "the first batch went before the pause noticed")
	assert.Equal(t, 5, book.kept("9.9.9.9"), "an interrupted revert can be run again")
}

func TestAnAccountIsRevertedWhateverScopeItTookFrom(t *testing.T) {
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "il", 3: "il"}}
	book.seed(tiles)
	book.record(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "campus", Account: guest, Country: "ps", Previous: "il"})
	book.record(clicks.WorldStage, ledger.Taking{Tile: 2, Scope: "home", Account: guest, Country: "ps", Previous: "il"})
	book.record(clicks.WorldStage, ledger.Taking{Tile: 3, Scope: "campus", Account: "a-classmate", Country: "ps", Previous: "il"})
	for tile := uint32(1); tile <= 3; tile++ {
		tiles.owners[tile] = "ps"
	}

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Account: guest})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Account: guest, Touched: 2, Held: 2, Restored: 2}, out)
	assert.Equal(t, map[uint32]string{1: "il", 2: "il", 3: "ps"}, tiles.owners, "the classmate on the same scope keeps its tile")

	again, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Account: guest})
	require.NoError(t, err)
	assert.Zero(t, again.Held, "the map holds none of its takes")
	assert.Equal(t, 1, book.kept("campus"), "only the account's takes are forgotten")
}

func TestItGivesTheBombersBlastBackToTheTilesStillEmpty(t *testing.T) {
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "il", 3: "ps"}}
	book.seed(tiles)
	book.record(clicks.WorldStage, ledger.Bombing{Scope: "9.9.9.9", Blast: clicks.Blast{
		Tile: 2, CountryID: "de", Cleared: []uint32{1, 2, 3}, Owners: []string{"il", "il", "ps"},
	}})
	tiles.owners = map[uint32]string{1: "", 2: "", 3: ""}
	book.take(tiles, 3, "1.1.1.1", "fr")

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 2, Held: 2, Restored: 2}, out)
	assert.Equal(t, map[uint32]string{1: "il", 2: "il", 3: "fr"}, tiles.owners, "a tile retaken since stays retaken")
}

func TestARevertWaitsForTheWorldsMapWhileAnArenaPlays(t *testing.T) {
	book, tiles := setup(t)
	tiles.playing = clicks.MapID("battle").On(clicks.TestBoard)

	_, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9", DryRun: true})

	require.ErrorIs(t, err, revert_player_usecase.ErrArenaPlays)
	assert.Equal(t, 5, book.kept("9.9.9.9"), "nothing forgotten")
}

func TestARevertAfterAnArenaUndoesNoneOfItsTakesNorAnotherBoards(t *testing.T) {
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "ps", 3: "ps"}}
	book.seed(tiles)
	at := time.Date(2026, 11, 11, 21, 0, 0, 0, time.UTC)
	book.record(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "9.9.9.9", Country: "ps", Previous: "il", At: at})
	book.record(clicks.MapID("battle").On(clicks.TestBoard),
		ledger.Taking{Tile: 2, Scope: "9.9.9.9", Country: "ps", Previous: "", At: at.Add(10 * time.Minute)})
	book.record(clicks.World.On(clicks.TestBoard+1),
		ledger.Taking{Tile: 3, Scope: "9.9.9.9", Country: "ps", Previous: "il", At: at.Add(20 * time.Minute)})
	tiles.owners[1] = "ps"

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, 1, out.Restored)
	assert.Equal(t, map[uint32]string{1: "il", 2: "ps", 3: "ps"}, tiles.owners,
		"the world's tile 2 keeps its flag: the arena's take of it was none of the world's; nor was tile 3 of another board")
}

type carry map[uint32]uint32

func (c carry) Each(visit func(child, parent uint32)) {
	for _, child := range slices.Sorted(maps.Keys(c)) {
		visit(child, c[child])
	}
}

func TestARevertAfterTheWorldSwitchedBoardsGivesBackWhatTheSwitchCarriedOverToo(t *testing.T) {
	earth2 := clicks.World.On(clicks.TestBoard + 1)
	book := newWorld(t)
	earth1 := &stubMap{owners: map[uint32]string{1: "lv", 9: "lv"}}
	book.seed(earth1)
	book.take(earth1, 1, "9.9.9.9", "ie")
	book.switches[len(book.entries)] = func(replayed ledger.Replayed) {
		replayed.Cut(time.Time{}, earth2.Board, carry{5: 1, 6: 9, 7: 1})
	}
	tiles := &stubMap{playing: earth2, owners: map[uint32]string{5: "ie", 6: "lv", 7: "ie"}}
	book.take(tiles, 6, "9.9.9.9", "ps")
	book.take(tiles, 7, "1.1.1.1", "fr")

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, 2, out.Restored)
	assert.Equal(t, map[uint32]string{5: "lv", 6: "lv", 7: "fr"}, tiles.owners,
		"earth-2's tile 5 came over from the scope's take of earth-1's tile 1; tile 7 too, but France took it since")
}

func TestARevertRunAgainGivesBackTheTakesTheOneBeforeCouldNotReach(t *testing.T) {
	book := newWorld(t)
	tiles := &stubMap{owners: map[uint32]string{1: "se", 2: "se"}}
	book.seed(tiles)
	book.take(tiles, 1, "9.9.9.9", "de")
	book.take(tiles, 2, "9.9.9.9", "de")
	book.Forget(ledger.Caller{Scope: "9.9.9.9"}, ledger.Position(len(book.entries)))
	tiles.owners[2] = "se"

	out, err := revert(book, tiles, 10).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 2, Held: 1, Restored: 1}, out)
	assert.Equal(t, map[uint32]string{1: "se", 2: "se"}, tiles.owners,
		"a revert of the ledger's last 72 hours forgot the take of tile 1 and left it on the map: run again, it goes")
}
