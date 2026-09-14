package memory_tile_storage_test

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/secondary/memory_tile_storage"
)

// legacySnapshot writes the pre-postgres file format by hand, so the test pins the format rather than the decoder.
func legacySnapshot(mapMaxIndex uint32, owners map[uint32]string) []byte {
	codes := []string{""}
	ids := map[string]uint16{"": 0}
	tiles := make([]uint16, mapMaxIndex+1)
	for tile, owner := range owners {
		id, ok := ids[owner]
		if !ok {
			id = uint16(len(codes))
			ids[owner] = id
			codes = append(codes, owner)
		}
		tiles[tile] = id
	}

	payload := binary.LittleEndian.AppendUint32(nil, mapMaxIndex)
	payload = binary.LittleEndian.AppendUint16(payload, uint16(len(codes)))
	for _, code := range codes {
		payload = append(payload, uint8(len(code)))
		payload = append(payload, code...)
	}
	for _, tile := range tiles {
		payload = binary.LittleEndian.AppendUint16(payload, tile)
	}

	raw := append([]byte("CPTILES\n"), 1)
	raw = binary.LittleEndian.AppendUint32(raw, crc32.ChecksumIEEE(payload))
	return append(raw, payload...)
}

func (s *testSuite) writeLegacySnapshot(raw []byte) string {
	path := filepath.Join(s.T().TempDir(), "tiles.snapshot")
	s.Require().NoError(os.WriteFile(path, raw, 0o600))
	return path
}

func (s *testSuite) TestALegacySnapshotIsImportedIntoAnEmptyTable() {
	ctx := context.Background()
	path := s.writeLegacySnapshot(legacySnapshot(maxIndex, map[uint32]string{1: "fr", 2: "us", maxIndex: "fr"}))
	persistence := newFakePersistence()
	storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, persistence)

	s.Require().NoError(storage.Load(ctx))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{1: "fr", 2: "us", maxIndex: "fr"}, state)
	s.InDelta(2.0/maxIndex, storage.Share("fr"), 1e-12)
	s.FileExists(path, "the file stays until postgres holds what it had")

	s.Require().NoError(storage.Flush(ctx))

	s.Equal(map[uint32]string{1: "fr", 2: "us", maxIndex: "fr"}, persistence.stored())
	s.NoFileExists(path)
	s.FileExists(path + ".imported")
}

func (s *testSuite) TestALegacySnapshotIsKeptWhileTheImportCannotBeWritten() {
	ctx := context.Background()
	path := s.writeLegacySnapshot(legacySnapshot(maxIndex, map[uint32]string{1: "fr"}))
	persistence := newFakePersistence()
	storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, persistence)
	s.Require().NoError(storage.Load(ctx))

	persistence.fail(os.ErrDeadlineExceeded)
	s.Require().Error(storage.Flush(ctx))

	s.FileExists(path, "a crash now must be able to import it again")
}

func (s *testSuite) TestALegacySnapshotIsIgnoredOncePostgresHoldsTiles() {
	path := s.writeLegacySnapshot(legacySnapshot(maxIndex, map[uint32]string{1: "fr"}))
	persistence := newFakePersistence()
	persistence.rows = map[uint32]string{5: "de"}
	storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, persistence)

	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{5: "de"}, state)
	s.FileExists(path)
}

func (s *testSuite) TestNoLegacySnapshotStartsEmpty() {
	path := filepath.Join(s.T().TempDir(), "does-not-exist.snapshot")
	storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, newFakePersistence())

	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, maxIndex)
	s.Require().NoError(err)
	s.Empty(state)
}

func (s *testSuite) TestALegacySnapshotOfAnEmptyMapIsRetiredAtOnce() {
	path := s.writeLegacySnapshot(legacySnapshot(maxIndex, nil))
	storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, newFakePersistence())

	s.Require().NoError(storage.Load(context.Background()))

	s.NoFileExists(path)
	s.FileExists(path + ".imported")
}

func (s *testSuite) TestALegacySnapshotOfABiggerMapImportsTheOverlap() {
	path := s.writeLegacySnapshot(legacySnapshot(1_000, map[uint32]string{10: "fr", 900: "us"}))
	storage := memory_tile_storage.New(100, memory_tile_storage.Config{LegacySnapshotPath: path},
		newFakePersistence(), slog.New(slog.DiscardHandler))

	s.Require().NoError(storage.Load(context.Background()))

	state, err := stateBatch(storage, 0, 100)
	s.Require().NoError(err)
	s.Equal(map[uint32]string{10: "fr"}, state)
}

func (s *testSuite) TestACorruptLegacySnapshotRefusesTheBoot() {
	valid := legacySnapshot(maxIndex, map[uint32]string{1: "fr"})

	corruptions := map[string][]byte{
		"empty file":          {},
		"shorter than header": valid[:4],
		"bad magic":           append([]byte("NOTATILE"), valid[8:]...),
		"truncated body":      valid[:len(valid)-10],
		"flipped byte": func() []byte {
			raw := append([]byte(nil), valid...)
			raw[len(raw)-1] ^= 0xff
			return raw
		}(),
		"unknown version": func() []byte {
			raw := append([]byte(nil), valid...)
			raw[8] = 99
			return raw
		}(),
	}

	for name, raw := range corruptions {
		s.Run(name, func() {
			path := s.writeLegacySnapshot(raw)
			storage := s.newStorageOn(memory_tile_storage.Config{LegacySnapshotPath: path}, newFakePersistence())

			s.Require().ErrorContains(storage.Load(context.Background()), "corrupt snapshot")
			s.FileExists(path)
		})
	}
}
