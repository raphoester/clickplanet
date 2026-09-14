package inmemory_ledger_storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"maps"
	"math"
	"os"
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
)

// Version 2 is a header, then frames appended one save at a time: the takes since the last save, and
// the marks (the oldest position kept, and each forgotten scope) when they moved. Each frame carries
// its own CRC32, so a crash mid-append costs the last frame and nothing before it. A file grown past
// twice what it keeps is written again whole.
//
// Version 1 held the last take per tile. It is read once, as one take per tile, and written over.
const (
	stateMagic  = "CPLEDGR\n"
	versionV1   = uint8(1)
	versionV2   = uint8(2)
	headerSize  = len(stateMagic) + 1
	frameHeader = 1 + 4 + 4
	recordSize  = 16

	frameTakes = uint8(1)
	frameMarks = uint8(2)

	compactSlack = 1 << 20
)

var errCorruptState = errors.New("corrupt ledger file")

// file is what the state file holds, as far as this process knows.
type file struct {
	// appendable is false until the file is known good, so the next save writes it whole.
	appendable   bool
	size         int
	saved        ledger.Position
	head         ledger.Position
	marksChanged bool
}

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
			s.saveOrLog()
		case <-ctx.Done():
			s.saveOrLog()
			return
		}
	}
}

func (s *Storage) saveOrLog() {
	if err := s.Save(); err != nil {
		s.logger.Error("failed to save the ledger",
			slog.String("path", s.config.StatePath),
			slog.Any("error", err),
		)
	}
}

// Save appends what changed since the last save, or writes the file whole when it cannot append.
func (s *Storage) Save() error {
	if s.config.StatePath == "" {
		return nil
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	s.mu.Lock()
	head := s.headPositionLocked()
	rewrite := !s.file.appendable || s.file.size > 2*s.liveLocked()*recordSize+compactSlack
	marks := rewrite || s.file.marksChanged || head != s.file.head
	end := s.next
	if !marks && end == s.file.saved {
		s.mu.Unlock()
		return nil
	}

	from := s.file.saved
	if rewrite {
		from = head
	}
	views := s.viewsLocked(from)
	forgotten := maps.Clone(s.forgotten)
	s.file.marksChanged = false
	s.mu.Unlock()

	var frames []byte
	if marks {
		frames = appendFrame(frames, frameMarks, encodeMarks(head, forgotten))
	}
	for _, v := range views {
		frames = appendFrame(frames, frameTakes, encodeTakes(v))
	}

	var err error
	if rewrite {
		frames = append([]byte(stateMagic+string(versionV2)), frames...)
		err = cpatomicfile.Write(s.config.StatePath, frames)
	} else {
		err = appendToFile(s.config.StatePath, frames)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		// A failed append can leave half a frame behind, and only a whole write gets rid of it.
		s.file.appendable = false
		s.file.marksChanged = true
		return fmt.Errorf("failed to write ledger file: %w", err)
	}

	if rewrite {
		s.file.size = 0
	}
	s.file.appendable = true
	s.file.size += len(frames)
	s.file.saved = end
	s.file.head = head

	return nil
}

func appendToFile(path string, data []byte) error {
	//nolint:gosec // G304: the path comes from config, never from a request.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("failed to open for append: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to append: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to sync: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close: %w", err)
	}

	return nil
}

func appendFrame(buf []byte, kind uint8, payload []byte) []byte {
	buf = append(buf, kind)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(payload))) //nolint:gosec // a chunk's worth of records.
	buf = binary.LittleEndian.AppendUint32(buf, crc32.ChecksumIEEE(payload))
	return append(buf, payload...)
}

func appendString(buf []byte, value string) []byte {
	value = value[:min(len(value), math.MaxUint16)]
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(value))) //nolint:gosec // clamped above.
	return append(buf, value...)
}

func encodeMarks(head ledger.Position, forgotten map[string]ledger.Position) []byte {
	buf := binary.LittleEndian.AppendUint64(nil, uint64(head))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(forgotten))) //nolint:gosec // one per reverted scope.
	for scope, before := range forgotten {
		buf = appendString(buf, scope)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(before))
	}
	return buf
}

// encodeTakes writes a view with a string table of its own, holding only the strings its records use.
func encodeTakes(v view) []byte {
	var scopes, countries []string
	scopeIDs := make(map[uint32]uint32)
	countryIDs := make(map[uint16]uint16)

	records := make([]byte, 0, recordSize*len(v.records))
	for _, r := range v.records {
		scope, ok := scopeIDs[r.scope]
		if !ok {
			scope = uint32(len(scopes)) //nolint:gosec // at most chunkSize.
			scopeIDs[r.scope] = scope
			scopes = append(scopes, v.scopes[r.scope])
		}

		ids := [2]uint16{}
		for i, id := range [2]uint16{r.country, r.previous} {
			local, ok := countryIDs[id]
			if !ok {
				local = uint16(len(countries)) //nolint:gosec // a chunk holds at most MaxUint16+1.
				countryIDs[id] = local
				countries = append(countries, v.countries[id])
			}
			ids[i] = local
		}

		records = binary.LittleEndian.AppendUint32(records, r.at)
		records = binary.LittleEndian.AppendUint32(records, r.tile)
		records = binary.LittleEndian.AppendUint32(records, scope)
		records = binary.LittleEndian.AppendUint16(records, ids[0])
		records = binary.LittleEndian.AppendUint16(records, ids[1])
	}

	buf := binary.LittleEndian.AppendUint64(nil, uint64(v.first))
	for _, table := range [][]string{scopes, countries} {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(table))) //nolint:gosec // at most chunkSize.
		for _, value := range table {
			buf = appendString(buf, value)
		}
	}
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(v.records))) //nolint:gosec // at most chunkSize.

	return append(buf, records...)
}

// LoadState fills the ledger from the state file. A missing or unreadable file starts empty, and a
// damaged one keeps the frames before the damage; either way it is written whole at the next save.
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

	s.mu.Lock()
	defer s.mu.Unlock()

	version, err := s.restoreLocked(raw)

	attrs := []any{slog.String("path", s.config.StatePath), slog.Int("version", int(version)), slog.Int("takings", s.liveLocked())}
	switch {
	case err != nil && s.liveLocked() == 0:
		s.logger.Error("failed to decode the ledger file, starting with an empty ledger", append(attrs, slog.Any("error", err))...)
	case err != nil:
		s.logger.Error("the ledger file is damaged, keeping the takes before the damage", append(attrs, slog.Any("error", err))...)
	default:
		s.logger.Info("restored the ledger", attrs...)
	}
}

func (s *Storage) restoreLocked(raw []byte) (uint8, error) {
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
		err := s.restoreV2Locked(raw[headerSize:])
		if err == nil {
			s.file = file{appendable: true, size: len(raw), saved: s.next, head: s.headPositionLocked()}
		}

		return version, err
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

	if s.headPositionLocked() < head {
		s.next = max(s.next, head)
		s.dropLocked(int(head - s.headPositionLocked())) //nolint:gosec // bounded by what was loaded.
	}

	s.forgotten = forgotten
	s.forgetMarksLocked()

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

		position := first + ledger.Position(i) //nolint:gosec // i < count.
		if position != s.next {
			// A gap in positions is takes dropped before a whole write: start a chunk at the new position.
			if n := len(s.chunks); n > 0 {
				s.sealLocked(s.chunks[n-1])
			}
			s.next = position
		}

		s.appendLocked(ledger.Taking{
			Tile:     tile,
			Scope:    scopes[scope],
			Country:  countries[country],
			Previous: countries[previous],
			At:       time.Unix(int64(at), 0).UTC(),
		})
	}

	return nil
}
