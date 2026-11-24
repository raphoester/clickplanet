package inmemory_tile_storage_test

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var setter = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x71}

func (s *testSuite) trapped(storage *inmemory_tile_storage.Storage, flag string, tiles ...uint32) {
	ctx := context.Background()
	for _, tile := range tiles {
		s.Require().NoError(storage.Set(ctx, tile, flag))
		_, err := clicks.NewTrapline(storage).Lay(ctx, clicks.NewTrap(setter, flag, tile))
		s.Require().NoError(err)
	}
}

func (s *testSuite) trapsOf(storage *inmemory_tile_storage.Storage, flag string) []uint32 {
	tiles, _ := storage.TrapsOf(flag)
	return tiles
}

func trapOn(tile uint32, flag string) clicks.TrapSnapshot {
	return clicks.TrapSnapshot{Tile: tile, Flag: flag, Account: setter}
}

func (s *testSuite) TestAFlushKeepsTheTrapsLaidAndABootReadsThemBack() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5, 9)
	s.trapped(storage, "de", 30)

	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.TrapSnapshot{trapOn(5, "fr"), trapOn(9, "fr"), trapOn(30, "de")}, persistence.StoredTrapsOn(clicks.World))
	booted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(booted.Load(ctx, clicks.WorldStage))
	s.Equal([]uint32{5, 9}, s.trapsOf(booted, "fr"))
	s.Equal([]uint32{30}, s.trapsOf(booted, "de"))
}

func (s *testSuite) TestATrapSprungOrBombedLeavesNoRowAfterTheNextFlush() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5, 9, 12)
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{5}})
	s.Require().NoError(err)
	_, err = clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(ctx, 12, "de")
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.TrapSnapshot{trapOn(9, "fr")}, persistence.StoredTrapsOn(clicks.World))
}

func (s *testSuite) TestAFailedFlushKeepsTheTrapsToSaveForTheNextOne() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5)
	persistence.FailWith(errors.New("connection reset"))
	s.Require().Error(storage.Flush(ctx))

	persistence.FailWith(nil)
	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.TrapSnapshot{trapOn(5, "fr")}, persistence.StoredTrapsOn(clicks.World))
}

func (s *testSuite) TestAWipeTakesTheTrapsWithTheMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5)
	s.Require().NoError(storage.Flush(ctx))

	wiped, err := storage.Wipe(ctx, keeps(true))
	s.Require().NoError(err)
	s.True(wiped)

	s.Empty(s.trapsOf(storage, "fr"))
	s.Empty(persistence.StoredTrapsOn(clicks.World))
}

func (s *testSuite) TestAnArenaPlaysWithNoTrapAndTheWorldsComeBackWithItsMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5)

	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, nil, opens(true))
	s.Require().NoError(err)
	s.Empty(s.trapsOf(storage, "fr"), "the arena's map holds none")

	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))
	s.Require().NoError(err)
	s.Equal([]uint32{5}, s.trapsOf(storage, "fr"))
}

func (s *testSuite) TestADeletedAccountsTrapsLeaveEveryMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.trapped(storage, "fr", 5, 9)
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.ForgetHolder(ctx, setter)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(s.trapsOf(storage, "fr"))
	s.Empty(persistence.StoredTrapsOn(clicks.World))
}

func (s *testSuite) TestAFlagKnockedOutLosesItsTrapsWithItsLand() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := clicks.NewTrapline(storage).Lay(ctx, clicks.NewTrap(setter, "fr", 5))
	s.Require().NoError(err)
	snare := clicks.NewSnareLog()

	_, err = clicks.NewClaiming(storage, tenShields, snare).Click(ctx, island[0], "de")
	s.Require().NoError(err)

	s.Empty(s.trapsOf(storage, "fr"))
	s.Empty(snare.Springs(), "the clear springs nothing")
}
