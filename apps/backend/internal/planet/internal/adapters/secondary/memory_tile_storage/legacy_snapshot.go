package memory_tile_storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"os"
)

// The pre-postgres snapshot format, read once to import it. Delete this file once production runs on postgres.

const (
	snapshotMagic   = "CPTILES\n"
	snapshotVersion = uint8(1)
	headerSize      = len(snapshotMagic) + 1 + 4
)

var errCorruptSnapshot = errors.New("corrupt snapshot")

const importedSuffix = ".imported"

// importLegacySnapshotLocked refuses a snapshot it cannot read: its map would be gone once a click lands in the empty table.
func (s *Storage) importLegacySnapshotLocked() error {
	path := s.config.LegacySnapshotPath
	if path == "" || !fileExists(path) {
		s.logger.Info("no stored tiles, starting from an empty map")
		return nil
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the legacy tile snapshot %s: %w", path, err)
	}

	codes, tiles, err := decodeSnapshot(raw)
	if err != nil {
		return fmt.Errorf("failed to import the legacy tile snapshot %s, move it away to start empty: %w", path, err)
	}

	if len(tiles) > len(s.tiles) {
		tiles = tiles[:len(s.tiles)]
	}

	owned := 0
	for tile, code := range tiles {
		if code == unownedCode {
			continue
		}
		if int(code) >= len(codes) {
			return fmt.Errorf("%w: tile %d references unknown country code %d", errCorruptSnapshot, tile, code)
		}

		id, err := s.internLocked(codes[code])
		if err != nil {
			return err
		}
		s.tiles[tile] = id
		s.counts[id]++
		s.markDirtyLocked(uint32(tile)) //nolint:gosec // tile <= maxIndex, which is a uint32.
		owned++
	}

	s.logger.Info("imported the legacy tile snapshot, it is renamed once postgres holds it",
		slog.String("path", path),
		slog.Int("ownedTiles", owned),
	)

	s.imported = path
	if owned == 0 {
		s.retireLegacySnapshot()
	}

	return nil
}

// retireLegacySnapshot renames the imported file, so a later boot on an emptied table cannot import it again.
func (s *Storage) retireLegacySnapshot() {
	if s.imported == "" {
		return
	}

	if err := os.Rename(s.imported, s.imported+importedSuffix); err != nil {
		s.logger.Error("failed to rename the imported tile snapshot", slog.String("path", s.imported), slog.Any("error", err))
	}

	s.imported = ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func decodeSnapshot(raw []byte) ([]string, []uint16, error) {
	if len(raw) < headerSize {
		return nil, nil, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptSnapshot, len(raw))
	}

	if string(raw[:len(snapshotMagic)]) != snapshotMagic {
		return nil, nil, fmt.Errorf("%w: bad magic", errCorruptSnapshot)
	}

	if version := raw[len(snapshotMagic)]; version != snapshotVersion {
		return nil, nil, fmt.Errorf("%w: unsupported version %d", errCorruptSnapshot, version)
	}

	payload := raw[headerSize:]
	want := binary.LittleEndian.Uint32(raw[len(snapshotMagic)+1:])
	if got := crc32.ChecksumIEEE(payload); got != want {
		return nil, nil, fmt.Errorf("%w: checksum mismatch (want %08x, got %08x)", errCorruptSnapshot, want, got)
	}

	if len(payload) < 6 {
		return nil, nil, fmt.Errorf("%w: truncated payload header", errCorruptSnapshot)
	}

	maxIndex := binary.LittleEndian.Uint32(payload)
	codeCount := int(binary.LittleEndian.Uint16(payload[4:]))
	payload = payload[6:]

	codes := make([]string, 0, codeCount)
	for i := 0; i < codeCount; i++ {
		if len(payload) < 1 {
			return nil, nil, fmt.Errorf("%w: truncated country code table", errCorruptSnapshot)
		}
		size := int(payload[0])
		if len(payload) < 1+size {
			return nil, nil, fmt.Errorf("%w: truncated country code", errCorruptSnapshot)
		}
		codes = append(codes, string(payload[1:1+size]))
		payload = payload[1+size:]
	}

	if len(codes) == 0 || codes[unownedCode] != "" {
		return nil, nil, fmt.Errorf("%w: country code table does not start with the empty code", errCorruptSnapshot)
	}

	tileCount := int(maxIndex) + 1
	if len(payload) != 2*tileCount {
		return nil, nil, fmt.Errorf("%w: expected %d bytes of tiles, got %d", errCorruptSnapshot, 2*tileCount, len(payload))
	}

	tiles := make([]uint16, tileCount)
	for i := range tiles {
		tiles[i] = binary.LittleEndian.Uint16(payload[2*i:])
	}

	return codes, tiles, nil
}
