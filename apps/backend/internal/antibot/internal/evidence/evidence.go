// Package evidence saves what the watchdogs and the jury are tracking, so a restart does not reset every window.
package evidence

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Config struct {
	// StatePath empty keeps the evidence in memory, where a restart empties it.
	StatePath    string
	SaveInterval time.Duration

	// Retention is the oldest evidence kept, on load and in memory, whatever a watchdog's own window says.
	Retention time.Duration
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

// Section is one watchdog's share of the file, or the jury's. Each encodes its own.
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

// Magic, version and a CRC32 of the payload, then the payload as gob.
const (
	stateMagic   = "CPEVIDN\n"
	stateVersion = uint8(1)
	headerSize   = len(stateMagic) + 1 + 4
)

var errCorruptState = errors.New("corrupt antibot evidence file")

type file struct {
	SavedAt  int64
	Sections map[string][]byte
}

func New(config Config, clock cptime.Clock, onStateError func(error), sections ...Section) *Store {
	if clock == nil {
		clock = cptime.SystemClock{}
	}
	if onStateError == nil {
		onStateError = func(error) {}
	}

	return &Store{
		config:       config.withDefaults(),
		clock:        clock,
		onStateError: onStateError,
		sections:     sections,
	}
}

type Store struct {
	config       Config
	clock        cptime.Clock
	onStateError func(error)
	sections     []Section

	mu      sync.Mutex
	savedAt time.Time // of the file loaded; zero when none was
}

// LoadState reads the evidence saved by the last process; a missing file starts empty, a bad one is reported and starts empty.
func (s *Store) LoadState() {
	if s.config.StatePath == "" {
		return
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	raw, err := os.ReadFile(s.config.StatePath)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		s.onStateError(fmt.Errorf("failed to read antibot evidence from %s: %w", s.config.StatePath, err))
		return
	}

	saved, err := decode(raw)
	if err != nil {
		s.onStateError(fmt.Errorf("failed to decode antibot evidence from %s, starting empty: %w", s.config.StatePath, err))
		return
	}

	before := s.clock.Now().Add(-s.config.Retention)

	for _, section := range s.sections {
		data, ok := saved.Sections[section.Name()]
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
	s.savedAt = time.Unix(0, saved.SavedAt)
	s.mu.Unlock()
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
			s.save()
		case <-ctx.Done():
			s.save()
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

func (s *Store) save() {
	if s.config.StatePath == "" {
		return
	}

	saved := file{
		SavedAt:  s.clock.Now().UnixNano(),
		Sections: make(map[string][]byte, len(s.sections)),
	}

	for _, section := range s.sections {
		data, err := section.Save()
		if err != nil {
			s.onStateError(fmt.Errorf("failed to encode %s evidence: %w", section.Name(), err))
			continue
		}
		saved.Sections[section.Name()] = data
	}

	raw, err := encode(saved)
	if err != nil {
		s.onStateError(fmt.Errorf("failed to encode antibot evidence: %w", err))
		return
	}

	if err := cpatomicfile.Write(s.config.StatePath, raw); err != nil {
		s.onStateError(fmt.Errorf("failed to save antibot evidence to %s: %w", s.config.StatePath, err))
	}
}

func encode(saved file) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, headerSize))
	if err := gob.NewEncoder(buf).Encode(saved); err != nil {
		return nil, fmt.Errorf("failed to encode: %w", err)
	}

	raw := buf.Bytes()
	copy(raw, stateMagic)
	raw[len(stateMagic)] = stateVersion
	binary.LittleEndian.PutUint32(raw[len(stateMagic)+1:], crc32.ChecksumIEEE(raw[headerSize:]))

	return raw, nil
}

func decode(raw []byte) (file, error) {
	if len(raw) < headerSize {
		return file{}, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptState, len(raw))
	}
	if string(raw[:len(stateMagic)]) != stateMagic {
		return file{}, fmt.Errorf("%w: bad magic", errCorruptState)
	}
	if version := raw[len(stateMagic)]; version != stateVersion {
		return file{}, fmt.Errorf("%w: unsupported version %d", errCorruptState, version)
	}

	payload := raw[headerSize:]
	want := binary.LittleEndian.Uint32(raw[len(stateMagic)+1:])
	if got := crc32.ChecksumIEEE(payload); got != want {
		return file{}, fmt.Errorf("%w: checksum mismatch (want %08x, got %08x)", errCorruptState, want, got)
	}

	var saved file
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&saved); err != nil {
		return file{}, fmt.Errorf("%w: %w", errCorruptState, err)
	}

	return saved, nil
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
