package restore_area_usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/restore_area_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	before = now.Add(-time.Hour)
)

type stubMap struct {
	owners  map[uint32]string
	batches []int
	err     error
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

type stubAreas struct {
	area []uint32
	err  error
}

func (a stubAreas) Around(uint32, uint32) ([]uint32, error) { return a.area, a.err }

func setup(t *testing.T) (*inmemory_ledger_storage.Storage, *stubMap) {
	t.Helper()

	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	tiles := &stubMap{owners: map[uint32]string{1: "fr", 2: "fr", 3: "", 4: "fr"}}

	book.Append(ledger.Taking{Tile: 1, Scope: "1.1.1.1", Country: "fr", At: before.Add(-time.Minute)})
	for tile := uint32(1); tile <= 3; tile++ {
		book.Append(ledger.Taking{Tile: tile, Scope: "9.9.9.9", Country: "de", Previous: tiles.owners[tile], At: before.Add(time.Minute)})
		tiles.owners[tile] = "de"
	}
	book.Append(ledger.Taking{Tile: 4, Scope: "9.9.9.9", Country: "de", Previous: "fr", At: before.Add(time.Minute)})
	tiles.owners[4] = "de"

	return book, tiles
}

func restore(book *inmemory_ledger_storage.Storage, tiles *stubMap, areas stubAreas, in restore_area_usecase.In) (restore_area_usecase.Out, error) {
	return restore_area_usecase.New(book, tiles, areas, ledger.NewReach(ledger.Config{Retention: 72 * time.Hour},
		cptime.NewFixedClock(now)), clicks.Pacing{Batch: 2}).Execute(context.Background(), in)
}

func TestTheAreaGoesBackToWhatItHeldBeforeInBatches(t *testing.T) {
	book, tiles := setup(t)

	out, err := restore(book, tiles, stubAreas{area: []uint32{1, 2, 3}}, restore_area_usecase.In{Tile: 2, Rings: 1, Before: before})
	require.NoError(t, err)

	assert.Equal(t, 3, out.Tiles)
	assert.Equal(t, 3, out.Changed)
	assert.Equal(t, 3, out.Restored)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "", 4: "de"}, tiles.owners, "a tile outside the area is left")
	assert.Equal(t, []int{2, 1}, tiles.batches)
	assert.Equal(t, now.Add(-72*time.Hour), out.KeptSince)
}

func TestItSaysWhoPaintedTheAreaSinceBeforeOnly(t *testing.T) {
	book, tiles := setup(t)

	out, err := restore(book, tiles, stubAreas{area: []uint32{1, 2, 3}}, restore_area_usecase.In{
		Tile: 2, Rings: 1, Before: before, DryRun: true,
	})
	require.NoError(t, err)

	require.Len(t, out.Painters, 1, "the take before the time is not the drawing")
	assert.Equal(t, 1, out.PaintersTotal)
	assert.Equal(t, "9.9.9.9", out.Painters[0].Scope)
	assert.Equal(t, 3, out.Painters[0].Takes, "its take outside the area is not counted")
	assert.Equal(t, "de", out.Painters[0].Flag)
}

func TestADryRunCountsAndRestoresNothing(t *testing.T) {
	book, tiles := setup(t)

	out, err := restore(book, tiles, stubAreas{area: []uint32{1, 2, 3}}, restore_area_usecase.In{
		Tile: 2, Rings: 1, Before: before, DryRun: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, out.Changed)
	assert.Zero(t, out.Restored)
	assert.Empty(t, tiles.batches)
}

func TestATileTakenSinceTheReadStaysTaken(t *testing.T) {
	book, tiles := setup(t)
	taker := &racingMap{stubMap: tiles, tile: 2, owner: "pl"}

	out, err := restore_area_usecase.New(book, taker, stubAreas{area: []uint32{1, 2, 3}},
		ledger.NewReach(ledger.Config{Retention: 72 * time.Hour}, cptime.NewFixedClock(now)), clicks.Pacing{Batch: 10}).
		Execute(t.Context(), restore_area_usecase.In{Tile: 2, Rings: 1, Before: before})
	require.NoError(t, err)

	assert.Equal(t, 2, out.Restored)
	assert.Equal(t, "pl", tiles.owners[2])
}

type racingMap struct {
	*stubMap
	tile  uint32
	owner string
}

func (m *racingMap) Restore(ctx context.Context, restorations []clicks.Restoration) (int, error) {
	m.owners[m.tile] = m.owner
	return m.stubMap.Restore(ctx, restorations)
}

func TestARestoreIsRefusedBeforeAnythingIsRead(t *testing.T) {
	book, tiles := setup(t)

	for name, refused := range map[string]struct {
		areas stubAreas
		in    restore_area_usecase.In
		err   error
	}{
		"no time":             {stubAreas{area: []uint32{1}}, restore_area_usecase.In{Tile: 1, Rings: 1}, restore_area_usecase.ErrNoTime},
		"past the ledger":     {stubAreas{area: []uint32{1}}, restore_area_usecase.In{Tile: 1, Rings: 1, Before: now.Add(-73 * time.Hour)}, restore_area_usecase.ErrPastTheLedger},
		"an area that is not": {stubAreas{err: clicks.ErrInvalidArea}, restore_area_usecase.In{Rings: 99, Before: before}, clicks.ErrInvalidArea},
	} {
		_, err := restore(book, tiles, refused.areas, refused.in)
		require.ErrorIs(t, err, refused.err, name)
	}
	assert.Empty(t, tiles.batches)
}

func TestAFailedRestoreSaysWhatItRestored(t *testing.T) {
	book, tiles := setup(t)
	tiles.err = errors.New("the map is closed")

	_, err := restore(book, tiles, stubAreas{area: []uint32{1, 2, 3}}, restore_area_usecase.In{Tile: 2, Rings: 1, Before: before})

	require.ErrorIs(t, err, tiles.err)
}
