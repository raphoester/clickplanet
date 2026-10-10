package enclose_click_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/enclose_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

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

type shields map[uint32]int

func (d shields) Shields(tile uint32) int { return d[tile] }

func (d shields) Strike(_ context.Context, tile uint32, _ string) bool {
	if d[tile] == 0 {
		return false
	}
	d[tile]--

	return true
}

type board struct {
	tiles
	shields
}

func (b board) Click(ctx context.Context, tile uint32, value string) error {
	return b.Set(ctx, tile, value)
}

func (b board) Fortify(context.Context, uint32, string, int) (clicks.Fortification, error) {
	return clicks.Fortification{}, clicks.ErrNotWhole
}

type rule struct {
	claiming clicks.Claiming
	err      error
}

func (r rule) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if r.err != nil {
		return click_usecase.Out{}, r.err
	}

	impact, err := r.claiming.Click(ctx, in.TileID, in.CountryID)

	return click_usecase.Out{Outcome: impact.Outcome}, err
}

type encloser struct{ claiming clicks.Claiming }

func (e encloser) Enclose(ctx context.Context, _ uint32, flag string, inside []uint32) error {
	for _, tile := range inside {
		if _, err := e.claiming.Claim(ctx, tile, flag); err != nil {
			return fmt.Errorf("failed to claim tile %d: %w", tile, err)
		}
	}

	return nil
}

type recorder struct{ published []bonuses.Enclosed }

func (r *recorder) PublishEnclosed(_ bonuses.Entrant, enclosed bonuses.Enclosed) {
	r.published = append(r.published, enclosed)
}

const caller bonuses.Holder = "a-player"

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

type fixture struct {
	grid      honeycomb
	tiles     tiles
	shields   shields
	charges   *inmemory_charge_storage.Storage
	published *recorder
	useCase   *enclose_click.UseCase
}

func setup(charged bool, err error) fixture {
	f := fixture{
		grid:    honeycomb{size: 12},
		tiles:   tiles{},
		shields: shields{},
		charges: inmemory_charge_storage.New(inmemory_charge_storage.Config{},
			bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 10},
			inmemory_charge_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler)),
		published: &recorder{},
	}
	if charged {
		f.charges.Grant(caller, bonuses.KindEncloseClicks, 1)
	}

	claiming := clicks.NewClaiming(board{tiles: f.tiles, shields: f.shields}, 10)
	f.useCase = enclose_click.New(rule{claiming: claiming, err: err}, f.charges,
		bonuses.NewTerrain(f.grid, f.tiles), enclose_click.NewAnnexer(encloser{claiming: claiming}, f.charges, f.published))

	return f
}

func (f fixture) click(t *testing.T, tile uint32) {
	t.Helper()

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: tile, CountryID: "fr", Enclose: true})
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
	assert.Zero(t, f.charges.Held(caller).Enclosures, "the charge is one shape, and this was it")
}

func TestAShapeTakesUnownedTilesAndEveryoneElsesAlike(t *testing.T) {
	f := setup(true, nil)
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

func TestAShieldedTileInsideAShapeLosesAShieldAndIsNotTaken(t *testing.T) {
	f := setup(true, nil)
	inner := []uint32{f.grid.id(5, 5), f.grid.id(6, 5)}
	f.own("de", inner...)
	f.shields[inner[0]] = 2
	closing := f.grid.id(4, 5)
	wall := append(f.grid.ring(5, 5), f.grid.ring(6, 5)...)
	for _, tile := range wall {
		if tile != inner[0] && tile != inner[1] && tile != closing {
			f.own("fr", tile)
		}
	}

	f.click(t, closing)

	assert.Equal(t, "de", f.tiles[inner[0]], "a bonus is never a way around the rule")
	assert.Equal(t, 1, f.shields[inner[0]])
	assert.Equal(t, "fr", f.tiles[inner[1]])
	require.Len(t, f.published.published, 1)
	assert.Zero(t, f.charges.Held(caller).Enclosures)
}

func TestAClickAShieldTookClosesNothing(t *testing.T) {
	f := setup(true, nil)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("fr", ring[1:]...)
	f.own("de", ring[0])
	f.shields[ring[0]] = 1

	f.click(t, ring[0])

	assert.Equal(t, "de", f.tiles[ring[0]])
	assert.Empty(t, f.tiles[centre], "the tile inside was not taken")
	assert.Empty(t, f.published.published)
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures, "a click that closes nothing keeps the charge")
}

func TestATriangleHasNoInsideAndCostsNothing(t *testing.T) {
	f := setup(true, nil)
	a, b, c := f.grid.id(5, 5), f.grid.id(6, 5), f.grid.id(5, 6)
	f.own("fr", a, b)

	f.click(t, c)

	assert.Empty(t, f.published.published)
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures, "a click that closes nothing keeps the charge")
}

func (f fixture) carve(closing uint32, hole []uint32) {
	for id := uint32(1); id <= uint32(f.grid.size*f.grid.size); id++ {
		f.tiles[id] = "fr"
	}
	for _, id := range append(hole, closing) {
		f.tiles[id] = ""
	}
}

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
	f.carve(f.grid.id(1, 4), append(f.row(1, 5, 10), f.grid.id(1, 6)))

	f.click(t, f.grid.id(1, 4))

	assert.Empty(t, f.published.published)
	assert.Empty(t, f.tiles[f.grid.id(1, 5)], "nothing inside was taken")
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures)
}

func TestAShapeOpenToTheEdgeOfTheLandIsNotClosed(t *testing.T) {
	f := setup(true, nil)
	hole := f.row(0, 5, 2)
	f.carve(f.grid.id(1, 4), hole)

	f.click(t, f.grid.id(1, 4))

	assert.Empty(t, f.published.published)
	assert.Empty(t, f.tiles[hole[0]])
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures)
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

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: ring[0], CountryID: "fr", Enclose: true})

	require.ErrorIs(t, err, refused)
	assert.Empty(t, f.tiles[centre])
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures)
}

func TestAClickWithEncloseSwitchedOffClosesNothingAndKeepsTheCharge(t *testing.T) {
	f := setup(true, nil)
	centre, ring := f.grid.id(5, 5), f.grid.ring(5, 5)
	f.own("de", centre)
	f.own("fr", ring[1:]...)

	_, err := f.useCase.Execute(played(t), click_usecase.In{TileID: ring[0], CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, "de", f.tiles[centre])
	assert.Empty(t, f.published.published)
	assert.Equal(t, 1, f.charges.Held(caller).Enclosures, "the charge is used only when the player chooses")
}
