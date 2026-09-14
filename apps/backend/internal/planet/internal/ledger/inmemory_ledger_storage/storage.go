package inmemory_ledger_storage

import (
	"log/slog"
	"maps"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Config struct {
	// Where the ledger is saved. Empty keeps it in memory, where a restart empties it.
	StatePath    string
	SaveInterval time.Duration
	// The most takes kept, oldest dropped first even inside the retention. 16 bytes each.
	MaxTakes int
}

const (
	defaultSaveInterval = time.Minute
	defaultMaxTakes     = 4_000_000

	// A chunk is 1 MiB of records. Takes are dropped from the front, so a whole chunk goes at once.
	chunkSize = 1 << 16
)

func (c Config) withDefaults() Config {
	if c.SaveInterval <= 0 {
		c.SaveInterval = defaultSaveInterval
	}
	if c.MaxTakes <= 0 {
		c.MaxTakes = defaultMaxTakes
	}
	return c
}

func New(config Config, logger *slog.Logger) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Storage{
		config:    config.withDefaults(),
		logger:    logger,
		forgotten: make(map[string]ledger.Position),
	}
}

// Storage is an append-only log of takes in chunks. A record is never changed once written, so a
// replay copies the chunk headers under the lock and reads the records without it.
type Storage struct {
	config Config
	logger *slog.Logger

	mu     sync.Mutex
	chunks []*chunk
	// head is how many records of chunks[0] are dropped.
	head      int
	next      ledger.Position
	forgotten map[string]ledger.Position
	// full is set while the cap drops takes, so it is reported once rather than per take.
	full bool

	saveMu sync.Mutex
	file   file
}

var _ ledger.Storage = (*Storage)(nil)

// record is 16 bytes. Strings are interned per chunk, so a dropped chunk takes its strings with it.
type record struct {
	at       uint32
	tile     uint32
	scope    uint32
	country  uint16
	previous uint16
}

type chunk struct {
	first     ledger.Position
	records   []record
	scopes    []string
	countries []string

	// Only the open chunk interns; a sealed one drops its maps.
	scopeIDs   map[string]uint32
	countryIDs map[string]uint16
}

func newChunk(first ledger.Position) *chunk {
	return &chunk{
		first:      first,
		records:    make([]record, 0, chunkSize),
		scopeIDs:   make(map[string]uint32),
		countryIDs: make(map[string]uint16),
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

func (c *chunk) internScope(value string) uint32 {
	id, ok := c.scopeIDs[value]
	if !ok {
		id = uint32(len(c.scopes)) //nolint:gosec // at most chunkSize scopes.
		c.scopeIDs[value] = id
		c.scopes = append(c.scopes, value)
	}

	return id
}

func (s *Storage) Append(taking ledger.Taking) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appendLocked(taking)
}

func (s *Storage) appendLocked(taking ledger.Taking) {
	open := s.openChunkLocked()

	country, fits := open.internCountry(taking.Country)
	previous, fitsToo := open.internCountry(taking.Previous)
	if !fits || !fitsToo {
		s.sealLocked(open)
		open = s.openChunkLocked()
		country, _ = open.internCountry(taking.Country)
		previous, _ = open.internCountry(taking.Previous)
	}

	open.records = append(open.records, record{
		at:       seconds(taking.At),
		tile:     taking.Tile,
		scope:    open.internScope(taking.Scope),
		country:  country,
		previous: previous,
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
		if last := s.chunks[n-1]; last.scopeIDs != nil && len(last.records) < chunkSize {
			return last
		}
		s.sealLocked(s.chunks[n-1])
	}

	open := newChunk(s.next)
	s.chunks = append(s.chunks, open)

	return open
}

func (s *Storage) sealLocked(c *chunk) {
	c.scopeIDs = nil
	c.countryIDs = nil
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

// dropLocked drops the n oldest takes. A chunk emptied is let go, never reused, since a replay may still read it.
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
	for scope, before := range s.forgotten {
		if before <= head {
			delete(s.forgotten, scope)
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

func (s *Storage) Forget(scope string, before ledger.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if before > s.forgotten[scope] && before > s.headPositionLocked() {
		s.forgotten[scope] = before
		s.file.marksChanged = true
	}
}

// view is a chunk as a replay reads it: its headers, copied under the lock.
type view struct {
	first     ledger.Position
	records   []record
	scopes    []string
	countries []string
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

		views = append(views, view{
			first:     c.first + ledger.Position(start), //nolint:gosec // at most chunkSize.
			records:   c.records[start:len(c.records):len(c.records)],
			scopes:    c.scopes[:len(c.scopes):len(c.scopes)],
			countries: c.countries[:len(c.countries):len(c.countries)],
		})
	}

	return views
}

func (s *Storage) Replay(see func(ledger.Taking)) ledger.Position {
	s.mu.Lock()
	views := s.viewsLocked(0)
	end := s.next
	forgotten := maps.Clone(s.forgotten)
	s.mu.Unlock()

	for _, v := range views {
		for i, r := range v.records {
			taking := v.taking(r)
			if before, ok := forgotten[taking.Scope]; ok && v.first+ledger.Position(i) < before { //nolint:gosec // i < chunkSize.
				continue
			}
			see(taking)
		}
	}

	return end
}

func (v view) taking(r record) ledger.Taking {
	return ledger.Taking{
		Tile:     r.tile,
		Scope:    v.scopes[r.scope],
		Country:  v.countries[r.country],
		Previous: v.countries[r.previous],
		At:       time.Unix(int64(r.at), 0).UTC(),
	}
}

// seconds is a time as the record keeps it: to the second, until 2106.
func seconds(at time.Time) uint32 {
	return uint32(min(max(at.Unix(), 0), math.MaxUint32)) //nolint:gosec // clamped.
}
