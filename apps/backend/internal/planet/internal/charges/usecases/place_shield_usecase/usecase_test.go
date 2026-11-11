package place_shield_usecase_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const caller charges.Holder = "a-player"

type fixture struct {
	wallet  *charges.MemoryStore
	kept    *placements.MemoryStore
	tiles   *inmemory_tile_storage.Storage
	gate    placements.Gate
	escrow  placements.Escrow
	useCase *place_shield_usecase.UseCase
}

func setup(t *testing.T, shields int) fixture {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	wallet := charges.NewMemoryStore()
	wallet.Keep(caller, charges.Held{Shields: shields})
	kept := placements.NewMemoryStore()
	escrow := placements.NewEscrow(cpatomic.FakeTransactioner{}, charges.NewCharges(wallet, twoATile), kept,
		&placements.SequentialIDs{}, clock)

	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), clicks.NoCapitals(), inmemory_tile_storage.Config{},
		inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{7: "fr", 8: "de"}), cpatomic.FakeTransactioner{}, escrow,
		clock, slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.World))

	book := allegiances.NewBook(allegiances.NewMemoryStore(), cpcountries.New(), func() allegiances.Lock { return false }, allegiances.EveryFlagStands{}, clock,
		&allegiances.RecordedPledges{})
	require.NoError(t, book.Pledge(t.Context(), clicks.Payer{Account: string(caller)}, "fr"))
	gate := placements.NewGate(tiles, cpcountries.New(), book, twoATile)

	return fixture{
		wallet: wallet, kept: kept, tiles: tiles, gate: gate, escrow: escrow,
		useCase: place_shield_usecase.New(gate, escrow, tiles),
	}
}

var twoATile = func() func() charges.Rules {
	rules := charges.TestRules()
	rules.Shield.Held, rules.Shield.PerTile = 12, 2
	return func() charges.Rules { return rules }
}()

func played(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), string(caller))
}

func (f fixture) place(t *testing.T, in place_shield_usecase.In) (charges.Held, error) {
	t.Helper()

	return f.useCase.Execute(played(t), in)
}

func TestAShieldStandsOnATileOfTheFlagAndIsUnderWayUntilTheFlush(t *testing.T) {
	f := setup(t, 3)

	held, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})
	require.NoError(t, err)

	assert.Equal(t, 2, held.Shields, "the answer is what is left in hand")
	assert.Equal(t, 1, f.tiles.Shields(7))
	assert.Len(t, f.kept.Kept(), 1)

	require.NoError(t, f.tiles.Flush(t.Context()))
	assert.Empty(t, f.kept.Kept(), "the flush that saved the shield ends its placement")
}

func TestAPlacementTheGateRefusesSpendsNothing(t *testing.T) {
	f := setup(t, 3)

	_, err := f.place(t, place_shield_usecase.In{TileID: 8, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotYourTile)
	assert.Equal(t, 3, f.wallet.Kept(caller).Shields)
	assert.Empty(t, f.kept.Kept())
}

func TestNoShieldHeldPlacesNothing(t *testing.T) {
	f := setup(t, 0)

	_, err := f.place(t, place_shield_usecase.In{TileID: 7, CountryID: "fr"})

	require.ErrorIs(t, err, place_shield_usecase.ErrNoShield)
	assert.Zero(t, f.tiles.Shields(7))
	assert.Empty(t, f.kept.Kept())
}

type lostRace struct{ *inmemory_tile_storage.Storage }

func (lostRace) Shield(context.Context, uint32, string, int, clicks.PlacementID) (int, error) {
	return 0, clicks.ErrNotYourTile
}

func TestAShieldTheMapRefusesComesBackEvenAboveTheStackSize(t *testing.T) {
	f := setup(t, 20)
	useCase := place_shield_usecase.New(f.gate, f.escrow, lostRace{f.tiles})

	_, err := useCase.Execute(played(t), place_shield_usecase.In{TileID: 7, CountryID: "fr"})

	require.ErrorIs(t, err, clicks.ErrNotYourTile, "the tile changed hands between the gate and the map")
	assert.Equal(t, 20, f.wallet.Kept(caller).Shields, "the stack size is 12, and the shield still comes back")
	assert.Empty(t, f.kept.Kept())
}
