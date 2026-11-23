package inmemory_tile_storage_test

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var sapper = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x61}

func (s *testSuite) mined(storage *inmemory_tile_storage.Storage, flag string, tiles ...uint32) {
	ctx := context.Background()
	for _, tile := range tiles {
		s.Require().NoError(storage.Set(ctx, tile, flag))
		_, err := clicks.NewMinefield(storage, line{}, tenShields, clicks.NewSnareLog()).Lay(ctx, clicks.NewMine(sapper, "203.0.113.9", false, flag, tile))
		s.Require().NoError(err)
	}
}

func (s *testSuite) minesOf(storage *inmemory_tile_storage.Storage, flag string) []uint32 {
	tiles, _ := storage.MinesOf(flag)
	return tiles
}

func mineOn(tile uint32, flag string) clicks.MineSnapshot {
	return clicks.MineSnapshot{Tile: tile, Flag: flag, Account: sapper, Scope: "203.0.113.9"}
}

func (s *testSuite) TestAFlushKeepsTheMinesLaidAndABootReadsThemBack() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5, 9)
	s.mined(storage, "de", 30)

	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.MineSnapshot{mineOn(5, "fr"), mineOn(9, "fr"), mineOn(30, "de")}, persistence.StoredMinesOn(clicks.World))
	booted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(booted.Load(ctx, clicks.WorldStage))
	s.Equal([]uint32{5, 9}, s.minesOf(booted, "fr"))
	s.Equal([]uint32{30}, s.minesOf(booted, "de"))
}

func (s *testSuite) TestAMineThatLeftTheMapLeavesNoRowAfterTheNextFlush() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5, 9)
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.Clear(ctx, clicks.Blast{Cleared: []uint32{5}})
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.MineSnapshot{mineOn(9, "fr")}, persistence.StoredMinesOn(clicks.World))
}

func (s *testSuite) TestAFailedFlushKeepsTheMinesToSaveForTheNextOne() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5)
	persistence.FailWith(errors.New("connection reset"))
	s.Require().Error(storage.Flush(ctx))

	persistence.FailWith(nil)
	s.Require().NoError(storage.Flush(ctx))

	s.Equal([]clicks.MineSnapshot{mineOn(5, "fr")}, persistence.StoredMinesOn(clicks.World))
}

func (s *testSuite) TestAWipeTakesTheMinesWithTheMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5)
	s.Require().NoError(storage.Flush(ctx))

	wiped, err := storage.Wipe(ctx, keeps(true))
	s.Require().NoError(err)
	s.True(wiped)

	s.Empty(s.minesOf(storage, "fr"))
	s.Empty(persistence.StoredMinesOn(clicks.World))
}

func (s *testSuite) TestAnArenaPlaysWithNoMineAndTheWorldsComeBackWithItsMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5)

	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)
	s.Empty(s.minesOf(storage, "fr"), "the arena's map holds none")

	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))
	s.Require().NoError(err)
	s.Equal([]uint32{5}, s.minesOf(storage, "fr"))
}

func (s *testSuite) TestADeletedAccountsMinesLeaveEveryMapWithNoBlast() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.mined(storage, "fr", 5, 9)
	s.Require().NoError(storage.Flush(ctx))

	_, err := storage.ForgetHolder(ctx, sapper)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(s.minesOf(storage, "fr"))
	s.Empty(persistence.StoredMinesOn(clicks.World))
	select {
	case <-storage.Fuse():
		s.Fail("no mine went off")
	default:
	}
}

func (s *testSuite) TestAFlagKnockedOutLosesItsMinesWithItsLandAndNoneGoesOff() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := clicks.NewMinefield(storage, line{}, tenShields, clicks.NewSnareLog()).Lay(ctx, clicks.NewMine(sapper, "203.0.113.9", false, "fr", 5))
	s.Require().NoError(err)

	_, err = clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Click(ctx, island[0], "de")
	s.Require().NoError(err)

	s.Empty(s.minesOf(storage, "fr"))
	detonations, err := clicks.NewMinefield(storage, line{}, tenShields, clicks.NewSnareLog()).Detonate(ctx, 2)
	s.Require().NoError(err)
	s.Empty(detonations)
}

func (s *testSuite) TestAMineAKnockedOutFlagsTakeTrippedGoesOffNoMore() {
	ctx := context.Background()
	storage := s.arenaWithFrance(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	_, err := clicks.NewMinefield(storage, line{}, tenShields, clicks.NewSnareLog()).Lay(ctx, clicks.NewMine(sapper, "203.0.113.9", false, "fr", 5))
	s.Require().NoError(err)

	_, err = clicks.NewClaiming(storage, tenShields, clicks.NewSnareLog()).Enclose(ctx, clicks.Enclosure{Country: "de"}, []uint32{5, island[0]})
	s.Require().NoError(err)

	detonations, err := clicks.NewMinefield(storage, line{}, tenShields, clicks.NewSnareLog()).Detonate(ctx, 2)
	s.Require().NoError(err)
	s.Empty(detonations, "France is out: none of its land comes back")
	s.Zero(storage.Held("fr"))
}
