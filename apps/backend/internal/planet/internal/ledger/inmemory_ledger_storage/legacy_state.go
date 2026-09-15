package inmemory_ledger_storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// The pre-postgres ledger file, read once to import it. Delete this file once production runs on postgres.
//
// Version 2 is a header, then frames: the takes of one save, and the marks (the oldest position kept, and
// each forgotten scope) when they moved. Each frame carries its own CRC32. Version 1 held the last take per tile.
const (
	stateMagic  = "CPLEDGR\n"
	versionV1   = uint8(1)
	versionV2   = uint8(2)
	headerSize  = len(stateMagic) + 1
	frameHeader = 1 + 4 + 4
	recordSize  = 16

	frameTakes = uint8(1)
	frameMarks = uint8(2)
)

var errCorruptState = errors.New("corrupt ledger file")

// importLegacyStateLocked loads the ledger file into an empty ledger; the next flush writes it and renames the file.
// A file it cannot read refuses the boot, since the first flush would make postgres the ledger without it.
func (s *Storage) importLegacyStateLocked() error {
	path := s.config.LegacyStatePath
	if path == "" || !fileExists(path) {
		s.logger.Info("no stored ledger, starting with an empty ledger")
		return nil
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the legacy ledger file %s: %w", path, err)
	}

	version, err := s.restoreFileLocked(raw)
	attrs := []any{slog.String("path", path), slog.Int("version", int(version)), slog.Int("takings", s.liveLocked())}
	switch {
	case err != nil && s.liveLocked() == 0:
		return fmt.Errorf("failed to import the legacy ledger file %s, move it away to start empty: %w", path, err)
	case err != nil:
		s.logger.Error("the legacy ledger file is damaged, importing the takes before the damage", append(attrs, slog.Any("error", err))...)
	default:
		s.logger.Info("imported the legacy ledger file, it is renamed once postgres holds it", attrs...)
	}

	s.imported = path
	for scope := range s.forgotten {
		s.dirtyScopes[scope] = struct{}{}
	}

	return nil
}

func (s *Storage) restoreFileLocked(raw []byte) (uint8, error) {
	if len(raw) < headerSize {
		return 0, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptState, len(raw))
	}
	if string(raw[:len(stateMagic)]) != stateMagic {
		return 0, fmt.Errorf("%w: bad magic", errCorruptState)
	}

	switch version := raw[len(stateMagic)]; version {
	case versionV1:
		takings, err := decodeV1(raw)
		if err != nil {
			return version, err
		}

		sort.SliceStable(takings, func(i, j int) bool { return takings[i].At.Before(takings[j].At) })
		for _, taking := range takings {
			s.appendLocked(taking)
		}

		return version, nil
	case versionV2:
		return version, s.restoreV2Locked(raw[headerSize:])
	default:
		return version, fmt.Errorf("%w: unsupported version %d", errCorruptState, version)
	}
}

type frame struct {
	kind    uint8
	payload []byte
}

// restoreV2Locked reads the marks of every good frame first, so takes already dropped are never loaded.
func (s *Storage) restoreV2Locked(raw []byte) error {
	frames, damage := splitFrames(raw)

	var head ledger.Position
	forgotten := make(map[string]ledger.Position)
	var good []frame
	for _, f := range frames {
		if f.kind == frameMarks {
			if err := decodeMarks(f.payload, &head, forgotten); err != nil {
				damage = err
				break
			}
		}
		good = append(good, f)
	}

	for _, f := range good {
		if f.kind != frameTakes {
			continue
		}
		if err := s.restoreTakesLocked(f.payload, head); err != nil {
			damage = err
			break
		}
	}

	s.applyMarksLocked(head, forgotten)

	return damage
}

func splitFrames(raw []byte) ([]frame, error) {
	var frames []frame
	for len(raw) > 0 {
		if len(raw) < frameHeader {
			return frames, fmt.Errorf("%w: truncated frame header", errCorruptState)
		}

		kind := raw[0]
		size := int(binary.LittleEndian.Uint32(raw[1:]))
		want := binary.LittleEndian.Uint32(raw[5:])
		if len(raw) < frameHeader+size {
			return frames, fmt.Errorf("%w: truncated frame", errCorruptState)
		}

		payload := raw[frameHeader : frameHeader+size]
		if got := crc32.ChecksumIEEE(payload); got != want {
			return frames, fmt.Errorf("%w: frame checksum mismatch (want %08x, got %08x)", errCorruptState, want, got)
		}
		if kind != frameTakes && kind != frameMarks {
			return frames, fmt.Errorf("%w: unknown frame kind %d", errCorruptState, kind)
		}

		frames = append(frames, frame{kind: kind, payload: payload})
		raw = raw[frameHeader+size:]
	}

	return frames, nil
}

// reader walks a frame's payload; the first short read sticks, so a decoder checks once at the end.
type reader struct {
	buf []byte
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil || len(r.buf) < n {
		r.err = fmt.Errorf("%w: truncated frame payload", errCorruptState)
		return make([]byte, n)
	}

	out := r.buf[:n]
	r.buf = r.buf[n:]

	return out
}

func (r *reader) uint16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *reader) uint32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }
func (r *reader) uint64() uint64 { return binary.LittleEndian.Uint64(r.take(8)) }
func (r *reader) string() string { return string(r.take(int(r.uint16()))) }

func decodeMarks(payload []byte, head *ledger.Position, forgotten map[string]ledger.Position) error {
	r := reader{buf: payload}

	*head = max(*head, ledger.Position(r.uint64()))
	count := r.uint32()
	for i := uint32(0); i < count && r.err == nil; i++ {
		scope := r.string()
		forgotten[scope] = max(forgotten[scope], ledger.Position(r.uint64()))
	}

	return r.err
}

func (s *Storage) restoreTakesLocked(payload []byte, head ledger.Position) error {
	r := reader{buf: payload}

	first := ledger.Position(r.uint64())
	tables := [2][]string{}
	for i := range tables {
		count := int(r.uint32())
		tables[i] = make([]string, 0, min(count, len(r.buf)/2))
		for j := 0; j < count && r.err == nil; j++ {
			tables[i] = append(tables[i], r.string())
		}
	}
	count := int(r.uint32())
	if r.err != nil {
		return r.err
	}
	if len(r.buf) != count*recordSize {
		return fmt.Errorf("%w: expected %d bytes of records, got %d", errCorruptState, count*recordSize, len(r.buf))
	}
	if first < s.next {
		return fmt.Errorf("%w: takes at %d overlap takes up to %d", errCorruptState, first, s.next)
	}

	scopes, countries := tables[0], tables[1]
	for i := range count {
		if first+ledger.Position(i) < head { //nolint:gosec // i < count.
			r.take(recordSize)
			continue
		}

		at, tile, scope := r.uint32(), r.uint32(), r.uint32()
		country, previous := int(r.uint16()), int(r.uint16())
		if int(scope) >= len(scopes) || country >= len(countries) || previous >= len(countries) {
			return fmt.Errorf("%w: string out of range", errCorruptState)
		}

		if err := s.restoreLocked(first+ledger.Position(i), ledger.Taking{ //nolint:gosec // i < count.
			Tile:     tile,
			Scope:    scopes[scope],
			Country:  countries[country],
			Previous: countries[previous],
			At:       time.Unix(int64(at), 0).UTC(),
		}); err != nil {
			return err
		}
	}

	return nil
}
