package memory_tile_storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
)

const (
	snapshotMagic   = "CPTILES\n"
	snapshotVersion = uint8(1)
	headerSize      = len(snapshotMagic) + 1 + 4
)

var errCorruptSnapshot = errors.New("corrupt snapshot")

func (s *Storage) Run(ctx context.Context) {
	if s.config.SnapshotPath == "" {
		s.logger.Warn("no snapshot path configured, tile state will not survive a restart")
		<-ctx.Done()
		return
	}

	if err := cpatomicfile.CheckWritable(s.config.SnapshotPath); err != nil {
		s.logger.Error("snapshot path is not writable, tile state will not survive a restart",
			slog.String("path", s.config.SnapshotPath),
			slog.Any("error", err),
		)
	}

	s.logger.Info("snapshotting tile state",
		slog.String("path", s.config.SnapshotPath),
		slog.Duration("interval", s.config.SnapshotInterval),
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
			slog.String("path", s.config.SnapshotPath),
			slog.Any("error", err),
		)
	}
}

func (s *Storage) Snapshot() error {
	if s.config.SnapshotPath == "" {
		return nil
	}

	payload, ok := s.encode()
	if !ok {
		return nil
	}

	if err := cpatomicfile.Write(s.config.SnapshotPath, payload); err != nil {
		s.tilesMu.Lock()
		s.dirty = true
		s.tilesMu.Unlock()

		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

func (s *Storage) encode() ([]byte, bool) {
	s.tilesMu.Lock()
	if !s.dirty {
		s.tilesMu.Unlock()
		return nil, false
	}
	s.dirty = false

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

func (s *Storage) restore() {
	if s.config.SnapshotPath == "" {
		return
	}

	raw, err := os.ReadFile(s.config.SnapshotPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.logger.Info("no tile snapshot found, starting from an empty map",
				slog.String("path", s.config.SnapshotPath),
			)
			return
		}

		s.logger.Error("failed to read tile snapshot, starting from an empty map",
			slog.String("path", s.config.SnapshotPath),
			slog.Any("error", err),
		)
		return
	}

	codes, tiles, err := decodeSnapshot(raw)
	if err != nil {
		s.logger.Error("failed to decode tile snapshot, starting from an empty map",
			slog.String("path", s.config.SnapshotPath),
			slog.Any("error", err),
		)
		return
	}

	if len(tiles) != len(s.tiles) {
		s.logger.Warn("tile snapshot was taken with a different map size, restoring the overlap",
			slog.Int("snapshotTiles", len(tiles)),
			slog.Int("configuredTiles", len(s.tiles)),
		)
		if len(tiles) > len(s.tiles) {
			tiles = tiles[:len(s.tiles)]
		}
	}

	owned := 0
	counts := make([]uint32, len(codes))
	for _, code := range tiles {
		if code == unownedCode {
			continue
		}
		if int(code) >= len(codes) {
			s.logger.Error("tile snapshot references an unknown country code, starting from an empty map",
				slog.Int("codeID", int(code)),
			)
			return
		}
		counts[code]++
		owned++
	}

	copy(s.tiles, tiles)
	s.counts = counts
	s.codes = codes
	s.codeIDs = make(map[string]uint16, len(codes))
	for id, code := range codes {
		s.codeIDs[code] = uint16(id)
	}

	s.logger.Info("restored tile state from snapshot",
		slog.String("path", s.config.SnapshotPath),
		slog.Int("ownedTiles", owned),
		slog.Int("countryCodes", len(codes)-1),
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
