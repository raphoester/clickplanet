package enclose_click_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// honeycomb is a patch of the map: a size×size parallelogram of hexagons in
// axial coordinates. A tile inside it has six neighbours; a tile on its rim has
// fewer, which is what the edge of the land looks like to the real map.
type honeycomb struct{ size int }

func (h honeycomb) id(q, r int) uint32 { return uint32(r*h.size + q + 1) }

func (h honeycomb) Neighbours(id uint32) []uint32 {
	q, r := int(id-1)%h.size, int(id-1)/h.size

	var around []uint32
	for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, -1}, {-1, 1}} {
		nq, nr := q+step[0], r+step[1]
		if nq >= 0 && nq < h.size && nr >= 0 && nr < h.size {
			around = append(around, h.id(nq, nr))
		}
	}

	return around
}

// ring is the six tiles around (q, r), in order around it.
func (h honeycomb) ring(q, r int) []uint32 {
	return []uint32{
		h.id(q+1, r), h.id(q+1, r-1), h.id(q, r-1),
		h.id(q-1, r), h.id(q-1, r+1), h.id(q, r+1),
	}
}

type tiles map[uint32]string

func (t tiles) Owner(tile uint32) (string, bool) { return t[tile], true }

func (t tiles) Set(_ context.Context, tile uint32, value string) error {
	t[tile] = value
	return nil
}

// rule stands in for the click rule, and writes the clicked tile the way it does.
type rule struct {
	tiles tiles
	err   error
}

func (r rule) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if r.err != nil {
		return click_usecase.Out{}, r.err
	}

	return click_usecase.Out{}, r.tiles.Set(ctx, in.TileID, in.CountryID)
}

type recorder struct{ published []bonuses.Enclosed }

func (r *recorder) PublishEnclosed(_ string, enclosed bonuses.Enclosed) {
	r.published = append(r.published, enclosed)
}

// silence hears the charges change and tells nobody.
type silence struct{}

func (silence) PublishCharges(bonuses.Holder, bonuses.Held) {}

// caller is the holder of a click made from the test's context: no account, and the scope that reads.
var caller = bonuses.HolderOf(clicks.PayerOf(context.Background()))

type fixture struct {
	grid      honeycomb
	tiles     tiles
	charges   *bonuses.Charges
	published *recorder
	useCase   *enclose_click.UseCase
}

func setup(charged bool, err error) fixture {
	f := fixture{
		grid:  honeycomb{size: 12},
		tiles: tiles{},
		charges: bonuses.NewCharges(bonuses.ChargesConfig{TTL: time.Hour, SpreadClicks: 8, EnclosureMaxTiles: 10},
			cptime.NewFixedClock(epoch), silence{}),
		published: &recorder{},
	}
	if charged {
		f.charges.Grant(caller, bonuses.KindEncloseClicks)
	}

	f.useCase = enclose_click.New(rule{tiles: f.tiles, err: err}, f.charges,
		bonuses.NewTerrain(f.grid, f.tiles), enclose_click.NewAnnexer(f.tiles, f.charges, f.published))

	return f
}

func (f fixture) click(t *testing.T, tile uint32) {
	t.Helper()

	_, err := f.useCase.Execute(t.Context(), click_usecase.In{TileID: tile, CountryID: "fr"})
	require.NoError(t, err)
}

func (f fixture) own(country string, ids ...uint32) {
	for _, id := range ids {
		f.tiles[id] = country
	}
}

func TestClosingARingTakesTheTileInsideIt(t *testing.T) {
	f := setup(true, nil)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("de", centre)
	f.own("fr", ring[1:]...)

	f.click(t, ring[0])

	assert.Equal(t, "fr", f.tiles[centre])
	require.Len(t, f.published.published, 1)

	enclosed := f.published.published[0]
	assert.Equal(t, "fr", enclosed.CountryID)
	assert.Equal(t, ring[0], enclosed.ClosingTile)
	assert.Equal(t, []uint32{centre}, enclosed.Filled)
	assert.ElementsMatch(t, ring, enclosed.Wall)
	assert.False(t, f.charges.Held(caller).Enclose, "the charge is one shape, and this was it")
}

func TestAShapeTakesUnownedTilesAndEveryoneElsesAlike(t *testing.T) {
	f := setup(true, nil)
	// Two tiles inside a ring of ten: one unowned, one somebody else's.
	inner := []uint32{f.grid.id(5, 5), f.grid.id(6, 5)}
	f.own("de", inner[1])
	closing := f.grid.id(4, 5)
	wall := append(f.grid.ring(5, 5), f.grid.ring(6, 5)...)
	for _, tile := range wall {
		if tile != inner[0] && tile != inner[1] && tile != closing {
			f.own("fr", tile)
		}
	}

	f.click(t, closing)

	assert.Equal(t, "fr", f.tiles[inner[0]])
	assert.Equal(t, "fr", f.tiles[inner[1]])
	require.Len(t, f.published.published, 1)
	assert.Equal(t, inner, f.published.published[0].Filled)
}

func TestATriangleHasNoInsideAndCostsNothing(t *testing.T) {
	f := setup(true, nil)
	// Three tiles that all touch each other.
	a, b, c := f.grid.id(5, 5), f.grid.id(6, 5), f.grid.id(5, 6)
	f.own("fr", a, b)

	f.click(t, c)

	assert.Empty(t, f.published.published)
	assert.True(t, f.charges.Held(caller).Enclose, "a click that closes nothing keeps the charge")
}

// carve owns the whole patch for "fr" except the hole and the tile that will
// close it, which is the simplest way to wall in a shape of any size.
func (f fixture) carve(closing uint32, hole []uint32) {
	for id := uint32(1); id <= uint32(f.grid.size*f.grid.size); id++ {
		f.tiles[id] = "fr"
	}
	for _, id := range append(hole, closing) {
		f.tiles[id] = ""
	}
}

// row is n tiles in a line from (q, r), eastwards.
func (f fixture) row(q, r, n int) []uint32 {
	ids := make([]uint32, 0, n)
	for i := range n {
		ids = append(ids, f.grid.id(q+i, r))
	}

	return ids
}

func TestAShapeHoldingExactlyTheLimitIsTaken(t *testing.T) {
	f := setup(true, nil)
	hole := f.row(1, 5, 10)
	f.carve(f.grid.id(1, 4), hole)

	f.click(t, f.grid.id(1, 4))

	require.Len(t, f.published.published, 1)
	assert.ElementsMatch(t, hole, f.published.published[0].Filled)
}

func TestAShapeBiggerThanTheLimitTakesNothingAndCostsNothing(t *testing.T) {
	f := setup(true, nil)
	// Eleven tiles, none of them on the rim.
	f.carve(f.grid.id(1, 4), append(f.row(1, 5, 10), f.grid.id(1, 6)))

	f.click(t, f.grid.id(1, 4))

	assert.Empty(t, f.published.published)
	assert.Empty(t, f.tiles[f.grid.id(1, 5)], "nothing inside was taken")
	assert.True(t, f.charges.Held(caller).Enclose)
}

func TestAShapeOpenToTheEdgeOfTheLandIsNotClosed(t *testing.T) {
	f := setup(true, nil)
	// The tip of a peninsula: the tile on the rim has the sea on one side.
	hole := f.row(0, 5, 2)
	f.carve(f.grid.id(1, 4), hole)

	f.click(t, f.grid.id(1, 4))

	assert.Empty(t, f.published.published)
	assert.Empty(t, f.tiles[hole[0]])
	assert.True(t, f.charges.Held(caller).Enclose)
}

func TestClickingTheOutlineOfAShapeAlreadyClosedTakesNothing(t *testing.T) {
	f := setup(true, nil)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("de", centre)
	f.own("fr", ring...)

	f.click(t, ring[3])

	assert.Equal(t, "de", f.tiles[centre])
	assert.Empty(t, f.published.published)
}

func TestOneClickClosingTwoShapesTakesTheFirstForItsOneCharge(t *testing.T) {
	f := setup(true, nil)
	// Two holes of one tile each, both touching the tile clicked.
	top, bottom := f.grid.id(5, 4), f.grid.id(4, 6)
	f.carve(f.grid.id(5, 5), []uint32{top, bottom})

	f.click(t, f.grid.id(5, 5))

	require.Len(t, f.published.published, 1)

	taken := 0
	for _, tile := range []uint32{top, bottom} {
		if f.tiles[tile] == "fr" {
			taken++
		}
	}
	assert.Equal(t, 1, taken, "only the shape that was paid for is filled")
}

func TestAChargeSpentClosesNothingMore(t *testing.T) {
	f := setup(true, nil)
	first, firstRing := f.grid.id(3, 3), f.grid.ring(3, 3)
	second, secondRing := f.grid.id(8, 8), f.grid.ring(8, 8)
	f.own("fr", firstRing[1:]...)
	f.own("fr", secondRing[1:]...)

	f.click(t, firstRing[0])
	f.click(t, secondRing[0])

	assert.Equal(t, "fr", f.tiles[first])
	assert.Empty(t, f.tiles[second], "one charge, one shape")
	assert.Len(t, f.published.published, 1)
}

func TestWithoutTheBonusAClickClosesNothing(t *testing.T) {
	f := setup(false, nil)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("fr", ring[1:]...)

	f.click(t, ring[0])

	assert.Empty(t, f.tiles[centre])
	assert.Empty(t, f.published.published)
}

func TestARefusedClickClosesNothing(t *testing.T) {
	refused := errors.New("unknown country")
	f := setup(true, refused)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("fr", ring...)

	_, err := f.useCase.Execute(t.Context(), click_usecase.In{TileID: ring[0], CountryID: "fr"})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, f.tiles[centre])
	assert.True(t, f.charges.Held(caller).Enclose)
}
