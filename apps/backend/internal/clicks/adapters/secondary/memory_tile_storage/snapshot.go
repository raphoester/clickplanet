package memory_tile_storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

// Snapshot file layout — a compact binary encoding, roughly 2 bytes per tile:
//
//	magic     8 bytes  "CPTILES\n"
//	version   1 byte
//	checksum  4 bytes  CRC32 (IEEE) of everything below
//	--- payload ---
//	maxIndex  4 bytes  uint32, little endian
//	codeCount 2 bytes  uint16, number of interned country codes
//	codes     codeCount x (1 byte length + that many bytes), index 0 is ""
//	tiles     (maxIndex+1) x 2 bytes, little endian interned code id
const (
	snapshotMagic   = "CPTILES\n"
	snapshotVersion = uint8(1)
	headerSize      = len(snapshotMagic) + 1 + 4
)

var errCorruptSnapshot = errors.New("corrupt snapshot")

// Run keeps the snapshot file up to date until ctx is cancelled, then writes a
// final snapshot. It is the durability loop for the memory driver; with no
// snapshot path configured it just waits for ctx and returns.
func (s *Storage) Run(ctx context.Context) {
	if s.config.SnapshotPath == "" {
		s.logger.Warning("no snapshot path configured, tile state will not survive a restart")
		<-ctx.Done()
		return
	}

	s.logger.Info("snapshotting tile state",
		lf.String("path", s.config.SnapshotPath),
		lf.Any("interval", s.config.SnapshotInterval),
	)

	ticker := time.NewTicker(s.config.SnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.snapshotIfDirty()
		case <-ctx.Done():
			s.snapshotIfDirty()
			return
		}
	}
}

func (s *Storage) snapshotIfDirty() {
	if err := s.Snapshot(); err != nil {
		s.logger.Error("failed to write tile snapshot",
			lf.String("path", s.config.SnapshotPath),
			lf.Err(err),
		)
	}
}

// Snapshot writes the current state to the configured snapshot file, atomically
// via a temp file and a rename. It is a no-op when nothing changed since the
// last write, or when no path is configured.
func (s *Storage) Snapshot() error {
	if s.config.SnapshotPath == "" {
		return nil
	}

	payload, ok := s.encode()
	if !ok {
		return nil
	}

	if err := writeFileAtomic(s.config.SnapshotPath, payload); err != nil {
		// Put the dirty flag back so the next tick retries instead of
		// silently leaving the state unsaved.
		s.tilesMu.Lock()
		s.dirty = true
		s.tilesMu.Unlock()

		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

// encode serializes the state and clears the dirty flag. It reports false when
// there was nothing new to write.
func (s *Storage) encode() ([]byte, bool) {
	s.tilesMu.Lock()
	if !s.dirty {
		s.tilesMu.Unlock()
		return nil, false
	}
	s.dirty = false

	// Copy under the lock, encode outside of it: writers are only held up for
	// the duration of a couple of memory copies.
	codes := make([]string, len(s.codes))
	copy(codes, s.codes)
	tiles := make([]uint16, len(s.tiles))
	copy(tiles, s.tiles)
	maxIndex := s.maxIndex
	s.tilesMu.Unlock()

	codesSize := 0
	for _, code := range codes {
		codesSize += 1 + len(code)
	}

	buf := make([]byte, headerSize, headerSize+4+2+codesSize+2*len(tiles))

	buf = binary.LittleEndian.AppendUint32(buf, maxIndex)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(codes)))
	for _, code := range codes {
		buf = append(buf, uint8(len(code)))
		buf = append(buf, code...)
	}
	for _, tile := range tiles {
		buf = binary.LittleEndian.AppendUint16(buf, tile)
	}

	copy(buf, snapshotMagic)
	buf[len(snapshotMagic)] = snapshotVersion
	binary.LittleEndian.PutUint32(buf[len(snapshotMagic)+1:], crc32.ChecksumIEEE(buf[headerSize:]))

	return buf, true
}

// restore loads the snapshot file into the storage. Every failure mode —
// absent, truncated, checksum mismatch, unknown version — logs and leaves the
// storage empty rather than preventing a start.
func (s *Storage) restore() {
	if s.config.SnapshotPath == "" {
		return
	}

	raw, err := os.ReadFile(s.config.SnapshotPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.logger.Info("no tile snapshot found, starting from an empty map",
				lf.String("path", s.config.SnapshotPath),
			)
			return
		}

		s.logger.Error("failed to read tile snapshot, starting from an empty map",
			lf.String("path", s.config.SnapshotPath),
			lf.Err(err),
		)
		return
	}

	codes, tiles, err := decodeSnapshot(raw)
	if err != nil {
		s.logger.Error("failed to decode tile snapshot, starting from an empty map",
			lf.String("path", s.config.SnapshotPath),
			lf.Err(err),
		)
		return
	}

	// A snapshot taken with a different gameMap.maxIndex still restores what
	// overlaps, so growing or shrinking the map does not throw the state away.
	if len(tiles) != len(s.tiles) {
		s.logger.Warning("tile snapshot was taken with a different map size, restoring the overlap",
			lf.Int("snapshotTiles", len(tiles)),
			lf.Int("configuredTiles", len(s.tiles)),
		)
		if len(tiles) > len(s.tiles) {
			tiles = tiles[:len(s.tiles)]
		}
	}

	owned := 0
	for _, code := range tiles {
		if code == unownedCode {
			continue
		}
		if int(code) >= len(codes) {
			s.logger.Error("tile snapshot references an unknown country code, starting from an empty map",
				lf.Int("codeID", int(code)),
			)
			return
		}
		owned++
	}

	// Only mutate the storage once the whole snapshot has been validated, so a
	// bad file can never leave a half-restored map behind.
	copy(s.tiles, tiles)
	s.codes = codes
	s.codeIDs = make(map[string]uint16, len(codes))
	for id, code := range codes {
		s.codeIDs[code] = uint16(id)
	}

	s.logger.Info("restored tile state from snapshot",
		lf.String("path", s.config.SnapshotPath),
		lf.Int("ownedTiles", owned),
		lf.Int("countryCodes", len(codes)-1),
	)
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

// writeFileAtomic writes data to a temp file in the destination directory,
// fsyncs it, then renames it over path. A crash mid-write leaves the previous
// snapshot intact.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	defer func() {
		// No-op once the rename succeeded, cleanup otherwise.
		_ = os.Remove(tmpName)
	}()

	// CreateTemp makes the file 0600; snapshots are not secret and backup jobs
	// may well run as another user.
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return syncDir(dir)
}

// syncDir flushes the rename itself, so the snapshot survives a power loss and
// not just a process crash.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open snapshot directory: %w", err)
	}
	defer func() { _ = d.Close() }()

	if err := d.Sync(); err != nil {
		return fmt.Errorf("failed to sync snapshot directory: %w", err)
	}

	return nil
}
