package place_defender_usecase_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_defender_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const caller bonuses.Holder = "a-player"

type fixture struct {
	charges *inmemory_charge_storage.Storage
	tiles   *inmemory_tile_storage.Storage
	useCase *place_defender_usecase.UseCase
}

func setup(t *testing.T, defenders int) fixture {
	t.Helper()

	charges := inmemory_charge_storage.New(inmemory_charge_storage.Config{},
		bonuses.ChargesConfig{Defenders: 12}, inmemory_charge_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	if defenders > 0 {
		charges.Grant(caller, bonuses.KindDefenders, defenders)
	}

	tiles := inmemory_tile_storage.New(100, inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{7: "fr", 8: "de"}), slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context()))

	return fixture{
		charges: charges,
		tiles:   tiles,
		useCase: place_defender_usecase.New(charges, tiles, cpcountries.New(), 2),
	}
}

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

func (f fixture) place(t *testing.T, in place_defender_usecase.In) (bonuses.Held, error) {
	t.Helper()

	return f.useCase.Execute(played(t), in)
}

func TestADefenderStandsOnATileOfTheFlag(t *testing.T) {
	f := setup(t, 3)

	held, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Defenders, "the answer is what is left in hand")
	assert.Equal(t, 1, f.tiles.Defenders(7))
}

func TestATileOfAnotherFlagIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, 3)

	for _, tile := range []uint32{8, 9} {
		_, err := f.place(t, place_defender_usecase.In{TileID: tile, CountryID: "fr"})
		require.ErrorIs(t, err, clicks.ErrNotYourTile, tile)
	}

	assert.Equal(t, 3, f.charges.Held(caller).Defenders)
}

func TestAFullTileIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(t, 3)
	in := place_defender_usecase.In{TileID: 7, CountryID: "fr"}
	for range 2 {
		_, err := f.place(t, in)
		require.NoError(t, err)
	}

	_, err := f.place(t, in)
	require.ErrorIs(t, err, clicks.ErrTileFull)

	assert.Equal(t, 1, f.charges.Held(caller).Defenders)
	assert.Equal(t, 2, f.tiles.Defenders(7))
}

func TestNoDefenderHeldPlacesNothing(t *testing.T) {
	f := setup(t, 0)

	_, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr"})
	require.ErrorIs(t, err, place_defender_usecase.ErrNoDefender)

	assert.Zero(t, f.tiles.Defenders(7))
}

func TestAMalformedRequestIsRefused(t *testing.T) {
	f := setup(t, 3)

	_, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "nowhere"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	_, err = f.place(t, place_defender_usecase.In{TileID: 101, CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)

	assert.Equal(t, 3, f.charges.Held(caller).Defenders)
}

func TestADudSpendsTheDefenderAndPlacesNothing(t *testing.T) {
	f := setup(t, 3)

	held, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr", Dud: true})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Defenders)
	assert.Zero(t, f.tiles.Defenders(7))
}

type lostRace struct{ *inmemory_tile_storage.Storage }

func (l lostRace) Reinforce(context.Context, uint32, string, int) error { return clicks.ErrNotYourTile }

func TestADefenderThatCouldNotBePlacedIsGivenBack(t *testing.T) {
	f := setup(t, 3)
	useCase := place_defender_usecase.New(f.charges, lostRace{f.tiles}, cpcountries.New(), 2)

	_, err := useCase.Execute(played(t), place_defender_usecase.In{TileID: 7, CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrNotYourTile, "the tile changed hands between the check and the placement")

	assert.Equal(t, 3, f.charges.Held(caller).Defenders)
}
