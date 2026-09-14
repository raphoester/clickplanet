package inmemory_ledger_storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
)

// The file is the tile snapshot's shape: magic, version and a CRC32 of the
// payload, then a table of every string once, then one fixed-size record per
// tile. A full map is ~6 MB this way, against ~25 MB of JSON lines.
const (
	stateMagic   = "CPLEDGR\n"
	stateVersion = uint8(1)
	headerSize   = len(stateMagic) + 1 + 4
	recordSize   = 4 + 4 + 4 + 4 + 8
)

var errCorruptState = errors.New("corrupt ledger file")

func (s *Storage) Name() string { return "ledger-storage" }

func (s *Storage) Run(ctx context.Context) {
	if s.config.StatePath == "" {
		s.logger.Warn("no ledger state path configured, the ledger will not survive a restart")
		<-ctx.Done()
		return
	}

	if err := cpatomicfile.CheckWritable(s.config.StatePath); err != nil {
		s.logger.Error("ledger state path is not writable, the ledger will not survive a restart",
			slog.String("path", s.config.StatePath),
			slog.Any("error", err),
		)
	}

	ticker := time.NewTicker(s.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.saveIfDirty()
		case <-ctx.Done():
			s.saveIfDirty()
			return
		}
	}
}

func (s *Storage) saveIfDirty() {
	if err := s.Save(); err != nil {
		s.logger.Error("failed to save the ledger",
			slog.String("path", s.config.StatePath),
			slog.Any("error", err),
		)
	}
}

// Save writes the ledger if it changed since the last save.
func (s *Storage) Save() error {
	if s.config.StatePath == "" {
		return nil
	}

	payload, ok := s.encode()
	if !ok {
		return nil
	}

	if err := cpatomicfile.Write(s.config.StatePath, payload); err != nil {
		s.mu.Lock()
		s.dirty = true
		s.mu.Unlock()

		return fmt.Errorf("failed to write ledger file: %w", err)
	}

	return nil
}

func (s *Storage) encode() ([]byte, bool) {
	s.mu.Lock()
	if !s.dirty {
		s.mu.Unlock()
		return nil, false
	}
	s.dirty = false

	takings := make([]ledger.Taking, 0, len(s.tiles))
	for _, taking := range s.tiles {
		takings = append(takings, taking)
	}
	s.mu.Unlock()

	var strs []string
	ids := make(map[string]uint32)
	intern := func(value string) uint32 {
		id, ok := ids[value]
		if !ok {
			id = uint32(len(strs))
			ids[value] = id
			strs = append(strs, value)
		}
		return id
	}

	records := make([]byte, 0, recordSize*len(takings))
	for _, taking := range takings {
		records = binary.LittleEndian.AppendUint32(records, taking.Tile)
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Scope))
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Country))
		records = binary.LittleEndian.AppendUint32(records, intern(taking.Previous))
		records = binary.LittleEndian.AppendUint64(records, uint64(taking.At.UnixNano()))
	}

	buf := make([]byte, headerSize, headerSize+4+len(records)+8*len(strs))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(strs)))
	for _, value := range strs {
		buf = binary.LittleEndian.AppendUint16(buf, uint16(min(len(value), math.MaxUint16)))
		buf = append(buf, value[:min(len(value), math.MaxUint16)]...)
	}
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(takings)))
	buf = append(buf, records...)

	copy(buf, stateMagic)
	buf[len(stateMagic)] = stateVersion
	binary.LittleEndian.PutUint32(buf[len(stateMagic)+1:], crc32.ChecksumIEEE(buf[headerSize:]))

	return buf, true
}

// LoadState fills the ledger from the state file; a missing or bad one is logged and leaves it empty.
func (s *Storage) LoadState() {
	if s.config.StatePath == "" {
		return
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	raw, err := os.ReadFile(s.config.StatePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.logger.Info("no ledger file found, starting with an empty ledger",
				slog.String("path", s.config.StatePath),
			)
			return
		}

		s.logger.Error("failed to read the ledger file, starting with an empty ledger",
			slog.String("path", s.config.StatePath),
			slog.Any("error", err),
		)
		return
	}

	tiles, err := decodeState(raw)
	if err != nil {
		s.logger.Error("failed to decode the ledger file, starting with an empty ledger",
			slog.String("path", s.config.StatePath),
			slog.Any("error", err),
		)
		return
	}

	s.mu.Lock()
	s.tiles = tiles
	s.mu.Unlock()

	s.logger.Info("restored the ledger",
		slog.String("path", s.config.StatePath),
		slog.Int("takings", len(tiles)),
	)
}

func decodeState(raw []byte) (map[uint32]ledger.Taking, error) {
	if len(raw) < headerSize {
		return nil, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptState, len(raw))
	}

	if string(raw[:len(stateMagic)]) != stateMagic {
		return nil, fmt.Errorf("%w: bad magic", errCorruptState)
	}

	if version := raw[len(stateMagic)]; version != stateVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", errCorruptState, version)
	}

	payload := raw[headerSize:]
	want := binary.LittleEndian.Uint32(raw[len(stateMagic)+1:])
	if got := crc32.ChecksumIEEE(payload); got != want {
		return nil, fmt.Errorf("%w: checksum mismatch (want %08x, got %08x)", errCorruptState, want, got)
	}

	if len(payload) < 4 {
		return nil, fmt.Errorf("%w: truncated string table", errCorruptState)
	}
	count := int(binary.LittleEndian.Uint32(payload))
	payload = payload[4:]

	strs := make([]string, 0, min(count, len(payload)/2))
	for range count {
		if len(payload) < 2 {
			return nil, fmt.Errorf("%w: truncated string table", errCorruptState)
		}
		size := int(binary.LittleEndian.Uint16(payload))
		if len(payload) < 2+size {
			return nil, fmt.Errorf("%w: truncated string", errCorruptState)
		}
		strs = append(strs, string(payload[2:2+size]))
		payload = payload[2+size:]
	}

	if len(payload) < 4 {
		return nil, fmt.Errorf("%w: truncated record count", errCorruptState)
	}
	records := int(binary.LittleEndian.Uint32(payload))
	payload = payload[4:]

	if len(payload) != records*recordSize {
		return nil, fmt.Errorf("%w: expected %d bytes of records, got %d", errCorruptState, records*recordSize, len(payload))
	}

	lookup := func(id uint32) (string, error) {
		if int(id) >= len(strs) {
			return "", fmt.Errorf("%w: string %d out of range", errCorruptState, id)
		}
		return strs[id], nil
	}

	tiles := make(map[uint32]ledger.Taking, records)
	for i := range records {
		record := payload[i*recordSize:]

		scope, err := lookup(binary.LittleEndian.Uint32(record[4:]))
		if err != nil {
			return nil, err
		}
		country, err := lookup(binary.LittleEndian.Uint32(record[8:]))
		if err != nil {
			return nil, err
		}
		previous, err := lookup(binary.LittleEndian.Uint32(record[12:]))
		if err != nil {
			return nil, err
		}

		taking := ledger.Taking{
			Tile:     binary.LittleEndian.Uint32(record),
			Scope:    scope,
			Country:  country,
			Previous: previous,
			At:       time.Unix(0, int64(binary.LittleEndian.Uint64(record[16:]))),
		}
		tiles[taking.Tile] = taking
	}

	return tiles, nil
}
