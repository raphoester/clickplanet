package inmemory_tile_storage_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type line struct{}

func (line) Neighbours(id uint32) []uint32 { return []uint32{id - 1, id + 1} }

var (
	carrier = cpsession.AccountID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f}
	noon    = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

// A French virus on tile 5, with German land on each side of it.
func (s *testSuite) infected(storage *inmemory_tile_storage.Storage) clicks.Contagion {
	ctx := context.Background()
	for _, tile := range []uint32{3, 4, 5, 6, 7} {
		s.Require().NoError(storage.Set(ctx, tile, "de"))
	}
	contagion := clicks.NewContagion(storage, line{}, func(int) int { return 0 }, func() int { return 10 }, clicks.NewSnareLog())
	_, _, err := contagion.Start(ctx, clicks.NewVirus("v-1", carrier, "203.0.113.7", false, "fr", 5), 32, noon)
	s.Require().NoError(err)
	return contagion
}

func (s *testSuite) TestAFlushKeepsTheLiveVirusesAndABootReadsThemBack() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	contagion := s.infected(storage)
	_, _, err := contagion.Spread(ctx, "v-1", 32, noon.Add(20*time.Second))
	s.Require().NoError(err)

	s.Require().NoError(storage.Flush(ctx))

	kept := persistence.StoredVirusesOn(clicks.World)
	s.Require().Len(kept, 1)
	s.Equal([]uint32{5, 4}, kept[0].Infected)
	booted := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(booted.Load(ctx, clicks.WorldStage))
	s.Equal(storage.Viruses(), booted.Viruses())
	batch, err := booted.StateBatchDense(0, 10)
	s.Require().NoError(err)
	s.Equal([]clicks.InfectedTile{{Tile: 4, Virus: "v-1"}, {Tile: 5, Virus: "v-1"}}, batch.Infected)
}

func (s *testSuite) TestAVirusThatEndedLeavesNoRow() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	contagion := s.infected(storage)
	s.Require().NoError(storage.Flush(ctx))

	_, _, err := contagion.Spread(ctx, "v-1", 1, noon)
	s.Require().NoError(err)
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(persistence.StoredVirusesOn(clicks.World))
}

func (s *testSuite) TestAFlushWithNoVirusChangedKeepsTheVirusesAsTheyAre() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.infected(storage)
	s.Require().NoError(storage.Flush(ctx))
	s.Require().NoError(persistence.SaveViruses(ctx, clicks.WorldStage, nil))

	s.Require().NoError(storage.Set(ctx, 50, "it"))
	s.Require().NoError(storage.Flush(ctx))

	s.Empty(persistence.StoredVirusesOn(clicks.World), "nothing about the viruses was written again")
}

func (s *testSuite) TestEveryChangeOfOwnerCuresATileABombsToo() {
	ctx := context.Background()
	storage := s.newStorage(inmemory_tile_storage.Config{})
	contagion := s.infected(storage)
	_, _, err := contagion.Spread(ctx, "v-1", 32, noon)
	s.Require().NoError(err)

	_, err = storage.Clear(ctx, clicks.Blast{Cleared: []uint32{4}})
	s.Require().NoError(err)

	s.Equal([]uint32{5}, storage.Viruses()[0].TakeSnapshot().Infected)
	batch, err := storage.StateBatchDense(0, 10)
	s.Require().NoError(err)
	s.Equal([]clicks.InfectedTile{{Tile: 5, Virus: "v-1"}}, batch.Infected)
}

func (s *testSuite) TestAWipeTakesTheVirusesWithTheMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.infected(storage)

	wiped, err := storage.Wipe(ctx, keeps(true))
	s.Require().NoError(err)
	s.True(wiped)

	s.Empty(storage.Viruses())
	s.Empty(persistence.StoredVirusesOn(clicks.World))
}

func (s *testSuite) TestAnArenaPlaysWithNoVirusAndTheWorldsComeBackWithItsMap() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{})
	storage := s.loadedOn(inmemory_tile_storage.Config{}, persistence)
	s.infected(storage)
	before := storage.Viruses()

	_, err := storage.Blank(ctx, battle.On(clicks.TestBoard), noStart, opens(true))
	s.Require().NoError(err)
	s.Empty(storage.Viruses(), "the arena's map holds none")
	s.Len(persistence.StoredVirusesOn(clicks.World), 1, "the world's is kept aside")

	_, err = storage.PutBack(ctx, battle, clicks.WorldStage, keeps(true))
	s.Require().NoError(err)
	s.Equal(before, storage.Viruses())
}
