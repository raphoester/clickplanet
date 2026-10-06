package inmemory_ledger_storage

import (
	"log/slog"
	"maps"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Config struct {
	FlushInterval time.Duration
	MaxTakes      int
}

const (
	defaultFlushInterval = time.Second
	defaultMaxTakes      = 4_000_000

	chunkSize = 1 << 16
)

func (c Config) withDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}
	if c.MaxTakes <= 0 {
		c.MaxTakes = defaultMaxTakes
	}
	return c
}

func New(config Config, persistence Persistence, logger *slog.Logger) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Storage{
		config:      config.withDefaults(),
		persistence: persistence,
		logger:      logger,
		forgotten:   make(map[ledger.Caller]ledger.Position),
		dirtyMarks:  cpcolls.NewSet[ledger.Caller](),
	}
}

// A record is never changed or reused once written: Replay reads them outside the lock.
type Storage struct {
	config      Config
	persistence Persistence
	logger      *slog.Logger

	mu        sync.Mutex
	chunks    []*chunk
	head      int
	next      ledger.Position
	forgotten map[ledger.Caller]ledger.Position
	full      bool

	flushMu    sync.Mutex
	saved      ledger.Position
	savedHead  ledger.Position
	dirtyMarks *cpcolls.Set[ledger.Caller]
}

var _ ledger.Storage = (*Storage)(nil)

type record struct {
	at       uint32
	tile     uint32
	scope    uint32
	account  uint32
	country  uint16
	previous uint16
	kind     uint8
}

type chunk struct {
	first     ledger.Position
	records   []record
	callers   []string
	countries []string
	kinds     []string
	payloads  []payload

	callerIDs  map[string]uint32
	countryIDs map[string]uint16
	kindIDs    map[string]uint8
}

type payload struct {
	position ledger.Position
	bytes    []byte
}

func newChunk(first ledger.Position) *chunk {
	return &chunk{
		first:      first,
		records:    make([]record, 0, chunkSize),
		callerIDs:  make(map[string]uint32),
		countryIDs: make(map[string]uint16),
		kindIDs:    make(map[string]uint8),
	}
}

func (c *chunk) internCountry(value string) (uint16, bool) {
	if id, ok := c.countryIDs[value]; ok {
		return id, true
	}
	if len(c.countries) > math.MaxUint16 {
		return 0, false
	}

	id := uint16(len(c.countries))
	c.countryIDs[value] = id
	c.countries = append(c.countries, value)

	return id, true
}

func (c *chunk) internKind(value string) (uint8, bool) {
	if id, ok := c.kindIDs[value]; ok {
		return id, true
	}
	if len(c.kinds) > math.MaxUint8 {
		return 0, false
	}

	id := uint8(len(c.kinds))
	c.kindIDs[value] = id
	c.kinds = append(c.kinds, value)

	return id, true
}

func (c *chunk) internCaller(value string) uint32 {
	id, ok := c.callerIDs[value]
	if !ok {
		id = uint32(len(c.callers)) //nolint:gosec // at most two per record.
		c.callerIDs[value] = id
		c.callers = append(c.callers, value)
	}

	return id
}

func (s *Storage) Append(event ledger.Event) {
	entry, err := event.Entry()
	if err != nil {
		s.logger.Error("the ledger cannot write an event down, so it is not kept", slog.Any("error", err))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.appendLocked(entry)
}

func (s *Storage) appendLocked(entry ledger.Entry) {
	open := s.openChunkLocked()

	country, fits := open.internCountry(entry.Country)
	previous, fitsToo := open.internCountry(entry.Previous)
	kind, fitsAlso := open.internKind(entry.Kind)
	if !fits || !fitsToo || !fitsAlso {
		s.sealLocked(open)
		open = s.openChunkLocked()
		country, _ = open.internCountry(entry.Country)
		previous, _ = open.internCountry(entry.Previous)
		kind, _ = open.internKind(entry.Kind)
	}

	if entry.Payload != nil {
		open.payloads = append(open.payloads, payload{position: s.next, bytes: entry.Payload})
	}
	open.records = append(open.records, record{
		at:       seconds(entry.At),
		tile:     entry.Tile,
		scope:    open.internCaller(entry.Scope),
		account:  open.internCaller(entry.Account),
		country:  country,
		previous: previous,
		kind:     kind,
	})
	s.next++

	if excess := s.liveLocked() - s.config.MaxTakes; excess > 0 {
		s.dropLocked(excess)
		if !s.full {
			s.full = true
			s.logger.Warn("the ledger is full, dropping its oldest takes before the retention does",
				slog.Int("maxTakes", s.config.MaxTakes))
		}
	}
}

func (s *Storage) openChunkLocked() *chunk {
	if n := len(s.chunks); n > 0 {
		if last := s.chunks[n-1]; last.callerIDs != nil && len(last.records) < chunkSize {
			return last
		}
		s.sealLocked(s.chunks[n-1])
	}

	open := newChunk(s.next)
	s.chunks = append(s.chunks, open)

	return open
}

func (s *Storage) sealLocked(c *chunk) {
	c.callerIDs = nil
	c.countryIDs = nil
	c.kindIDs = nil
}

func (s *Storage) liveLocked() int {
	return int(s.next - s.headPositionLocked()) //nolint:gosec // bounded by MaxTakes.
}

func (s *Storage) headPositionLocked() ledger.Position {
	if len(s.chunks) == 0 {
		return s.next
	}

	return s.chunks[0].first + ledger.Position(s.head) //nolint:gosec // head < chunkSize.
}

func (s *Storage) dropLocked(n int) {
	for n > 0 && len(s.chunks) > 0 {
		first := s.chunks[0]
		step := min(n, len(first.records)-s.head)
		s.head += step
		n -= step

		if s.head < len(first.records) {
			break
		}
		s.chunks[0] = nil
		s.chunks = s.chunks[1:]
		s.head = 0
	}

	s.forgetMarksLocked()
}

func (s *Storage) forgetMarksLocked() {
	head := s.headPositionLocked()
	for caller, before := range s.forgotten {
		if before <= head {
			delete(s.forgotten, caller)
		}
	}
}

func (s *Storage) ForgetBefore(cutoff time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() { s.full = s.liveLocked() >= s.config.MaxTakes }()

	limit := seconds(cutoff)
	n := 0
	for _, c := range s.chunks {
		start := 0
		if c == s.chunks[0] {
			start = s.head
		}

		for _, r := range c.records[start:] {
			if r.at >= limit {
				s.dropLocked(n)
				return
			}
			n++
		}
	}

	s.dropLocked(n)
}

func (s *Storage) Forget(caller ledger.Caller, before ledger.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if before > s.forgotten[caller] && before > s.headPositionLocked() {
		s.forgotten[caller] = before
		s.dirtyMarks.Add(caller)
	}
}

type view struct {
	first     ledger.Position
	records   []record
	callers   []string
	countries []string
	kinds     []string
	payloads  []payload
}

func (s *Storage) viewsLocked(from ledger.Position) []view {
	views := make([]view, 0, len(s.chunks))
	for i, c := range s.chunks {
		start := 0
		if i == 0 {
			start = s.head
		}
		if end := c.first + ledger.Position(len(c.records)); end <= from { //nolint:gosec // at most chunkSize.
			continue
		}
		if from > c.first+ledger.Position(start) { //nolint:gosec // at most chunkSize.
			start = int(from - c.first) //nolint:gosec // inside this chunk.
		}

		first := c.first + ledger.Position(start) //nolint:gosec // at most chunkSize.
		payloads := c.payloads[:len(c.payloads):len(c.payloads)]
		for len(payloads) > 0 && payloads[0].position < first {
			payloads = payloads[1:]
		}

		views = append(views, view{
			first:     first,
			records:   c.records[start:len(c.records):len(c.records)],
			callers:   c.callers[:len(c.callers):len(c.callers)],
			countries: c.countries[:len(c.countries):len(c.countries)],
			kinds:     c.kinds[:len(c.kinds):len(c.kinds)],
			payloads:  payloads,
		})
	}

	return views
}

func (s *Storage) Replay(see func(ledger.Event)) ledger.Position {
	s.mu.Lock()
	views := s.viewsLocked(0)
	end := s.next
	forgotten := maps.Clone(s.forgotten)
	s.mu.Unlock()

	for _, v := range views {
		payloads := v.payloads
		for i := range v.records {
			var stored Stored
			stored, payloads = v.stored(i, payloads)
			if forgottenAt(forgotten, stored.Entry, stored.Position) {
				continue
			}

			event, err := ledger.EventOf(stored.Entry)
			if err != nil {
				s.logger.Error("an event of the ledger does not read back, so it is not replayed",
					slog.Uint64("position", uint64(stored.Position)), slog.Any("error", err))
				continue
			}
			see(event)
		}
	}

	return end
}

func forgottenAt(forgotten map[ledger.Caller]ledger.Position, entry ledger.Entry, position ledger.Position) bool {
	if before, ok := forgotten[ledger.Caller{Scope: entry.Scope}]; ok && position < before {
		return true
	}
	if entry.Account == "" {
		return false
	}
	before, ok := forgotten[ledger.Caller{Account: entry.Account}]

	return ok && position < before
}

func (v view) stored(i int, payloads []payload) (Stored, []payload) {
	r := v.records[i]
	position := v.first + ledger.Position(i) //nolint:gosec // i < chunkSize.

	entry := ledger.Entry{
		Kind:     v.kinds[r.kind],
		Tile:     r.tile,
		Scope:    v.callers[r.scope],
		Account:  v.callers[r.account],
		Country:  v.countries[r.country],
		Previous: v.countries[r.previous],
		At:       time.Unix(int64(r.at), 0).UTC(),
	}
	if len(payloads) > 0 && payloads[0].position == position {
		entry.Payload, payloads = payloads[0].bytes, payloads[1:]
	}

	return Stored{Position: position, Entry: entry}, payloads
}

func seconds(at time.Time) uint32 {
	return uint32(min(max(at.Unix(), 0), math.MaxUint32)) //nolint:gosec // clamped.
}
