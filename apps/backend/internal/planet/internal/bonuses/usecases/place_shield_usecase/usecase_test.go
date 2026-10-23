package place_shield_usecase_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const caller bonuses.Holder = "a-player"

type fixture struct {
	charges *bonuses.MemoryStore
	tiles   *inmemory_tile_storage.Storage
	book    allegiances.Book
	clock   *cptime.FixedClock
	useCase *place_shield_usecase.UseCase
}

func setup(t *testing.T, shields int) fixture {
	t.Helper()

	charges := bonuses.NewMemoryStore()
	charges.Keep(caller, bonuses.Held{Shields: shields})

	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{7: "fr", 8: "de"}), slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context()))

	clock := cptime.NewFixedClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	book := allegiances.NewBook(allegiances.NewMemoryStore(), cpcountries.New(), clock)

	return fixture{
		charges: charges,
		tiles:   tiles,
		book:    book,
		clock:   clock,
		useCase: place_shield_usecase.New(bonuses.NewCharges(charges, bonuses.ChargesConfig{Shields: 12}), tiles, cpcountries.New(), book, 2),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

func (f fixture) place(t *testing.T, in place_shield_usecase.In) (bonuses.Held, error) {
	t.Helper()

	return f.useCase.Execute(played(t), in)
}

func TestAShieldStandsOnATileOfTheFlag(t *testing.T) {
	f := setup(t, 3)

	held, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Shields, "the answer is what is left in hand")
	assert.Equal(t, 1, f.tiles.Shields(7))
}

func TestATileOfAnotherFlagIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, 3)

	for _, tile := range []uint32{8, 9} {
		_, err := f.place(t, place_shield_usecase.In{TileID: tile, CountryID: "fr"})
		require.ErrorIs(t, err, clicks.ErrNotYourTile, tile)
	}

	assert.Equal(t, 3, f.charges.Kept(caller).Shields)
}

func TestAFullTileIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, 3)
	in := place_shield_usecase.In{TileID: 7, CountryID: "fr"}
	for range 2 {
		_, err := f.place(t, in)
		require.NoError(t, err)
	}

	_, err := f.place(t, in)
	require.ErrorIs(t, err, clicks.ErrTileFull)

	assert.Equal(t, 1, f.charges.Kept(caller).Shields)
	assert.Equal(t, 2, f.tiles.Shields(7))
}

func TestNoShieldHeldPlacesNothing(t *testing.T) {
	f := setup(t, 0)

	_, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})
	require.ErrorIs(t, err, place_shield_usecase.ErrNoShield)

	assert.Zero(t, f.tiles.Shields(7))
}

func TestAMalformedRequestIsRefused(t *testing.T) {
	f := setup(t, 3)

	_, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "nowhere"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	_, err = f.place(t, place_shield_usecase.In{TileID: 101, CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)

	assert.Equal(t, 3, f.charges.Kept(caller).Shields)
}

type lostRace struct{ *inmemory_tile_storage.Storage }

func (l lostRace) Shield(context.Context, uint32, string, int) error { return clicks.ErrNotYourTile }

func TestAShieldThatCouldNotBePlacedIsGivenBack(t *testing.T) {
	f := setup(t, 3)
	useCase := place_shield_usecase.New(bonuses.NewCharges(f.charges, bonuses.ChargesConfig{Shields: 12}), lostRace{f.tiles}, cpcountries.New(), f.book, 2)

	_, err := useCase.Execute(played(t), place_shield_usecase.In{TileID: 7, CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrNotYourTile, "the tile changed hands between the check and the placement")

	assert.Equal(t, 3, f.charges.Kept(caller).Shields)
}

func TestAShieldForAnotherFlagWithinTheHourIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, 3)
	_, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)
	_, err = f.place(t, place_shield_usecase.In{TileID: 8, CountryID: "de"})
	require.NoError(t, err, "the first change is free")

	f.clock.Advance(30 * time.Minute)
	_, err = f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})

	require.ErrorIs(t, err, allegiances.ErrChangedTooSoon)
	assert.Equal(t, 1, f.charges.Kept(caller).Shields)
	assert.Equal(t, 1, f.tiles.Shields(7))
}
