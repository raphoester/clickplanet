package inmemory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func ownerIn(batch clicks.DenseBatch, tile uint32) string {
	return batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[(tile-batch.Start)*2:])]
}

func (s *testSuite) TestTheWorldsMapKeptAsideIsReadWhileAnArenaPlays() {
	ctx := context.Background()
	storage, _ := s.worldBeforeABattle(inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{}))
	world, err := storage.StateBatchDense(1, math.MaxUint32)
	s.Require().NoError(err)
	_, err = storage.Blank(ctx, battle, noStart, keeps(true))
	s.Require().NoError(err)
	s.Require().NoError(storage.Set(ctx, 12, "it"))

	kept, err := storage.Kept(ctx, clicks.World)
	s.Require().NoError(err)

	s.Equal(uint32(1), kept.Start)
	s.Len(kept.Tiles, len(world.Tiles), "the whole map")
	for _, tile := range append([]uint32{12, 13}, island...) {
		s.Equal(ownerIn(world, tile), ownerIn(kept, tile), "tile %d", tile)
	}
	s.Equal(world.Shields, kept.Shields, "the island's fortify")
	owner, _ := storage.Owner(12)
	s.Equal("it", owner, "the arena's map still plays")
}
