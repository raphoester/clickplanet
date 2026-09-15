// Package evidence saves what the watchdogs and the jury are tracking, so a restart does not reset every window.
package evidence

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	SaveInterval time.Duration

	// Retention is the oldest evidence kept, on load and in memory, whatever a watchdog's own window says.
	Retention time.Duration

	// LegacyStatePath is the evidence file from before postgres, imported once into an empty table.
	LegacyStatePath string
}

const (
	defaultSaveInterval = time.Minute
	defaultRetention    = 72 * time.Hour
)

func (c Config) withDefaults() Config {
	if c.SaveInterval <= 0 {
		c.SaveInterval = defaultSaveInterval
	}
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	return c
}

// Section is one watchdog's share of the evidence, or the jury's. Each encodes its own.
type Section interface {
	Name() string
	Save() ([]byte, error)
	// Load replaces what the section holds, or leaves it untouched on error.
	Load(data []byte) error
	Forget(before time.Time)
}

// Resumer is a section that reads gaps, and needs to know the outage to not read one there.
type Resumer interface {
	Resume(outage detect.Outage)
}

// Snapshot is every section's evidence as one flush wrote it.
type Snapshot struct {
	SavedAt  time.Time
	Sections map[string][]byte
}

// Persistence is where the evidence is kept between boots. It is never read after Load.
type Persistence interface {
	// Load answers an empty snapshot when nothing is stored.
	Load(ctx context.Context) (Snapshot, error)
	// Save replaces every stored section with the snapshot's.
	Save(ctx context.Context, snapshot Snapshot) error
}

const flushTimeout = 10 * time.Second

func New(config Config, clock cptime.Clock, persistence Persistence, onStateError func(error), sections ...Section) *Store {
	return &Store{
		config:       config.withDefaults(),
		clock:        clock,
		persistence:  persistence,
		onStateError: onStateError,
		sections:     sections,
	}
}

type Store struct {
	config       Config
	clock        cptime.Clock
	persistence  Persistence
	onStateError func(error)
	sections     []Section

	mu       sync.Mutex
	savedAt  time.Time // of the snapshot loaded; zero when none was
	imported string    // the legacy file loaded, renamed after the first flush
}

// Load refuses the boot when postgres cannot be read. A section that does not decode is reported and starts empty alone.
func (s *Store) Load(ctx context.Context) error {
	snapshot, err := s.persistence.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to read stored antibot evidence: %w", err)
	}

	if len(snapshot.Sections) > 0 {
		if legacyFileExists(s.config.LegacyStatePath) {
			s.onStateError(fmt.Errorf("the legacy evidence file %s is still on disk but postgres already holds evidence, ignoring it",
				s.config.LegacyStatePath))
		}
	} else {
		snapshot, err = s.importLegacyState()
		if err != nil {
			return err
		}
	}

	before := s.clock.Now().Add(-s.config.Retention)

	for _, section := range s.sections {
		data, ok := snapshot.Sections[section.Name()]
		if !ok {
			continue
		}
		if err := section.Load(data); err != nil {
			s.onStateError(fmt.Errorf("failed to load %s evidence, it starts empty: %w", section.Name(), err))
			continue
		}
		section.Forget(before)
	}

	s.mu.Lock()
	s.savedAt = snapshot.SavedAt
	s.mu.Unlock()

	return nil
}

// Resume tells the sections the process is now watching: the outage ends here, not at load, since the boot is not over then.
func (s *Store) Resume() {
	s.mu.Lock()
	savedAt := s.savedAt
	s.mu.Unlock()

	if savedAt.IsZero() {
		return
	}

	outage := detect.Outage{From: savedAt, To: s.clock.Now()}
	for _, section := range s.sections {
		if resumer, ok := section.(Resumer); ok {
			resumer.Resume(outage)
		}
	}
}

func (s *Store) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.forget()
			s.flushOrReport(ctx)
		case <-ctx.Done():
			s.flushOrReport(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *Store) forget() {
	before := s.clock.Now().Add(-s.config.Retention)
	for _, section := range s.sections {
		section.Forget(before)
	}
}

func (s *Store) flushOrReport(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := s.Flush(ctx); err != nil {
		s.onStateError(fmt.Errorf("failed to flush antibot evidence, retrying next tick: %w", err))
	}
}

// Flush writes every section as it is now, in one transaction. A section that fails to encode is reported and left out.
func (s *Store) Flush(ctx context.Context) error {
	snapshot := Snapshot{
		SavedAt:  s.clock.Now(),
		Sections: make(map[string][]byte, len(s.sections)),
	}

	for _, section := range s.sections {
		data, err := section.Save()
		if err != nil {
			s.onStateError(fmt.Errorf("failed to encode %s evidence: %w", section.Name(), err))
			continue
		}
		snapshot.Sections[section.Name()] = data
	}

	if err := s.persistence.Save(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save %d sections: %w", len(snapshot.Sections), err)
	}

	s.retireLegacyState()

	return nil
}

// Encode and Decode are the gob every section writes its own share with.
func Encode(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}
	return buf.Bytes(), nil
}

func Decode(data []byte, value any) error {
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(value); err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}
	return nil
}

// Nanos and Time are how every section writes a timestamp: eight bytes, no zone.
func Nanos(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func Time(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}
