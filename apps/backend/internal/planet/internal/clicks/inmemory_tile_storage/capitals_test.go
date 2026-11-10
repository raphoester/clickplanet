package inmemory_tile_storage_test

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var capitals = clicks.NewCapitals(map[string]uint32{"fr": island[0], "de": 20})

func capitalsWith(shields int) []clicks.StartTile {
	return []clicks.StartTile{{Tile: island[0], Country: "fr", Shields: shields}, {Tile: 20, Country: "de", Shields: shields}}
}

func (s *testSuite) withCapitals(persistence *inmemory_tile_storage.MemoryPersistence) *inmemory_tile_storage.Storage {
	storage := inmemory_tile_storage.New(borders, capitals, inmemory_tile_storage.Config{}, persistence,
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{},
		slog.New(slog.DiscardHandler))
	s.Require().NoError(storage.Load(context.Background(), clicks.World))
	return storage
}

func (s *testSuite) TestABlankStartsTheArenaWithItsCapitalsKeptWithIt() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{12: "de"})
	storage := s.withCapitals(persistence)

	blanked, err := storage.Blank(ctx, battle, capitalsWith(100), keeps(true))

	s.Require().NoError(err)
	s.True(blanked)
	s.Equal(map[uint32]string{island[0]: "fr", 20: "de"}, persistence.StoredOn(battle), "before any flush")
	s.Equal(map[uint32]int{island[0]: 100, 20: 100}, persistence.StoredShieldsOn(battle))
	owner, _ := storage.Owner(20)
	s.Equal("de", owner)
	s.Equal(100, storage.Shields(20))
	s.Equal(1, storage.Held("fr"))
	s.Equal(map[uint32]string{12: "de"}, persistence.Stored(), "the world's map has no capital")
}

func (s *testSuite) TestABlankThatCannotBeKeptStartsNoCapital() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.withCapitals(persistence)

	blanked, err := storage.Blank(ctx, battle, capitalsWith(100), keeps(false))

	s.Require().NoError(err)
	s.False(blanked)
	s.Empty(persistence.StoredOn(battle))
	owner, _ := storage.Owner(20)
	s.Empty(owner)
}

func (s *testSuite) TestACapitalTakesNoShieldWhileAnArenaPlays() {
	ctx := context.Background()
	storage := s.withCapitals(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := storage.Blank(ctx, battle, capitalsWith(0), keeps(true))
	s.Require().NoError(err)

	_, err = storage.Shield(ctx, 20, "de", 10, clicks.PlacementID{})
	s.Require().ErrorIs(err, clicks.ErrCapital)
	s.Zero(storage.Shields(20))

	s.Require().NoError(storage.Set(ctx, 21, "de"))
	_, err = storage.Shield(ctx, 21, "de", 10, clicks.PlacementID{})
	s.Require().NoError(err, "a tile that is no capital still takes one")
}

func (s *testSuite) TestACapitalTakesShieldsOnTheWorldsMap() {
	ctx := context.Background()
	storage := s.withCapitals(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{20: "de"}))

	_, err := storage.Shield(ctx, 20, "de", 10, clicks.PlacementID{})
	s.Require().NoError(err)
	s.Equal(1, storage.Shields(20))
}

func (s *testSuite) TestAFortifyLeavesAnArenasCapitalAsItStarted() {
	ctx := context.Background()
	storage := s.withCapitals(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := storage.Blank(ctx, battle, capitalsWith(0), keeps(true))
	s.Require().NoError(err)
	for _, tile := range island[1:] {
		s.Require().NoError(storage.Set(ctx, tile, "fr"))
	}

	fortification, err := storage.Fortify(ctx, island[2], "fr", 10)

	s.Require().NoError(err)
	s.Equal([]clicks.TileShields{{Tile: island[1], Shields: 1}, {Tile: island[2], Shields: 1}}, fortification.Raised)
	s.Zero(storage.Shields(island[0]))
}

func (s *testSuite) TestTheMapSaysWhetherItIsAnArenas() {
	ctx := context.Background()
	storage := s.withCapitals(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))

	world, err := storage.StateBatchDense(1, 100)
	s.Require().NoError(err)
	_, err = storage.Blank(ctx, battle, noStart, keeps(true))
	s.Require().NoError(err)
	arena, err := storage.StateBatchDense(1, 100)
	s.Require().NoError(err)

	s.False(world.Arena)
	s.True(arena.Arena)
}

func (s *testSuite) TestHoldsAnySaysWhetherAFlagHoldsOneOfTheTiles() {
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{5: "fr"}))
	s.Require().NoError(storage.Load(context.Background(), clicks.World))

	s.True(storage.HoldsAny("fr", []uint32{4, 5}))
	s.False(storage.HoldsAny("fr", []uint32{4}))
	s.False(storage.HoldsAny("de", []uint32{5}))
	s.False(storage.HoldsAny("fr", []uint32{maxIndex + 1}), "off the map")
	s.False(storage.HoldsAny("fr", []uint32{}))
}
