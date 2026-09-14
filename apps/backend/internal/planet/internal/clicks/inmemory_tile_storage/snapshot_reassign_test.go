package inmemory_tile_storage_test

import (
	"context"
	"path/filepath"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
)

func (s *testSuite) TestReassignedTilesSurviveASnapshot() {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	ctx := context.Background()

	storage := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})
	s.Require().NoError(storage.Set(ctx, 12, "dz"))
	_, _, err := storage.Reassign(ctx, "dz", "fr", 0, 10)
	s.Require().NoError(err)
	s.Require().NoError(storage.Snapshot())

	restored := s.newStorage(inmemory_tile_storage.Config{SnapshotPath: path})
	owner, _ := restored.Owner(12)
	s.Equal("fr", owner)
	s.Equal(1, restored.Held("fr"))
	s.Zero(restored.Held("dz"))
}
