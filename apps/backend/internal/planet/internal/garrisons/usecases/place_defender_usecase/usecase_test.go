package place_defender_usecase_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/inmemory_garrison_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons/usecases/place_defender_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const caller bonuses.Holder = "a-player"

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return o[tile], true
}

type fixture struct {
	charges   *inmemory_charge_storage.Storage
	garrisons *inmemory_garrison_storage.Storage
	useCase   *place_defender_usecase.UseCase
}

func setup(defenders int) fixture {
	charges := inmemory_charge_storage.New(inmemory_charge_storage.Config{},
		bonuses.ChargesConfig{Defenders: 12}, inmemory_charge_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	if defenders > 0 {
		charges.Grant(caller, bonuses.KindDefenders, defenders)
	}
	garrisons := inmemory_garrison_storage.New(inmemory_garrison_storage.Config{}, 2,
		inmemory_garrison_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))

	return fixture{
		charges:   charges,
		garrisons: garrisons,
		useCase:   place_defender_usecase.New(charges, owners{7: "fr", 8: "de"}, garrisons, cpcountries.New()),
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
	f := setup(3)

	held, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Defenders, "the answer is what is left in hand")
	assert.Equal(t, 1, f.garrisons.Defenders(7, "fr"))
}

func TestATileOfAnotherFlagIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(3)

	for _, tile := range []uint32{8, 9} {
		_, err := f.place(t, place_defender_usecase.In{TileID: tile, CountryID: "fr"})
		require.ErrorIs(t, err, place_defender_usecase.ErrNotYours, tile)
	}

	assert.Equal(t, 3, f.charges.Held(caller).Defenders)
}

func TestAFullTileIsRefusedAndSpendsNothing(t *testing.T) {
	f := setup(3)
	in := place_defender_usecase.In{TileID: 7, CountryID: "fr"}
	for range 2 {
		_, err := f.place(t, in)
		require.NoError(t, err)
	}

	_, err := f.place(t, in)
	require.ErrorIs(t, err, place_defender_usecase.ErrFull)

	assert.Equal(t, 1, f.charges.Held(caller).Defenders)
	assert.Equal(t, 2, f.garrisons.Defenders(7, "fr"))
}

func TestNoDefenderHeldPlacesNothing(t *testing.T) {
	f := setup(0)

	_, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr"})
	require.ErrorIs(t, err, place_defender_usecase.ErrNoDefender)

	assert.Zero(t, f.garrisons.Defenders(7, "fr"))
}

func TestAMalformedRequestIsRefused(t *testing.T) {
	f := setup(3)

	_, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "nowhere"})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)

	_, err = f.place(t, place_defender_usecase.In{TileID: 101, CountryID: "fr"})
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)

	_, err = f.place(t, place_defender_usecase.In{TileID: 9, CountryID: ""})
	require.ErrorIs(t, err, clicks.ErrUnknownCountry, "an empty tile wears no flag, not the empty one")

	assert.Equal(t, 3, f.charges.Held(caller).Defenders)
}

func TestADudSpendsTheDefenderAndPlacesNothing(t *testing.T) {
	f := setup(3)

	held, err := f.place(t, place_defender_usecase.In{TileID: 7, CountryID: "fr", Dud: true})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Defenders)
	assert.Zero(t, f.garrisons.Defenders(7, "fr"))
}
