package inmemory_tile_storage_test

import (
	"context"
	"encoding/binary"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func shieldsOf(tally []clicks.FlagShields) map[string]uint32 {
	read := map[string]uint32{}
	for _, flag := range tally {
		read[flag.Flag] = flag.Shields
	}
	return read
}

func (s *testSuite) shieldsOnTheTiles(storage *inmemory_tile_storage.Storage) map[string]uint32 {
	batch, err := storage.StateBatchDense(0, maxIndex)
	s.Require().NoError(err)

	shields := map[string]uint32{}
	for _, tile := range batch.Shields {
		owner := batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[2*(tile.Tile-batch.Start):])]
		shields[owner] += uint32(tile.Shields) //nolint:gosec // a tile's shields fit a byte.
	}
	return shields
}

func (s *testSuite) TestTheShieldTallyIsTheShieldsOnEachFlagsTiles() {
	ctx := context.Background()
	persistence := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de", island[0]: "fr", island[1]: "fr"})
	persistence.Shield(1, 3)
	persistence.Shield(3, 2)
	storage := s.newStorageOn(inmemory_tile_storage.Config{}, persistence)
	s.Require().NoError(storage.Load(ctx, clicks.WorldStage))
	s.Equal(map[string]uint32{"fr": 3, "de": 2}, shieldsOf(storage.ShieldTally()), "booted from what postgres keeps")

	_, err := storage.Shield(ctx, 2, "fr", 10, clicks.PlacementID{})
	s.Require().NoError(err)
	s.Require().NoError(storage.Claim(ctx, func(tiles clicks.Claimable) error {
		return tiles.Set(ctx, clicks.TileSnapshot{ID: 1, Owner: "fr", Shields: 2}.Restore())
	}))
	_, err = storage.Clear(ctx, clicks.Blast{Cleared: []uint32{2, 3}})
	s.Require().NoError(err)
	s.Equal(map[string]uint32{"fr": 2, "de": 1}, shieldsOf(storage.ShieldTally()),
		"one placed on tile 2, one struck on tile 1, and one each by the bomb")

	s.Require().NoError(storage.Click(ctx, 4, "de"))
	s.Require().NoError(storage.Click(ctx, 2, "de"))
	s.Equal(map[string]uint32{"fr": 2, "de": 1}, shieldsOf(storage.ShieldTally()), "takes of tiles with no shield")

	s.Require().NoError(storage.Set(ctx, island[2], "fr"))
	_, err = storage.Fortify(ctx, island[0], "fr", 10)
	s.Require().NoError(err)
	_, err = storage.Restore(ctx, []clicks.Restoration{{Tile: 1, From: "fr", To: "it"}})
	s.Require().NoError(err)
	s.Equal(map[string]uint32{"de": 1, "fr": 3}, shieldsOf(storage.ShieldTally()),
		"the fortify shields each tile of the island, and an operator's write sends tile 1's shields home")

	s.Equal(s.shieldsOnTheTiles(storage), shieldsOf(storage.ShieldTally()))
}
