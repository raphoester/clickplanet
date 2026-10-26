package listener

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var (
	_ evidence.Section = (*Marks)(nil)
	_ evidence.Holder  = (*Marks)(nil)
)

func NewMarks() *Marks {
	return &Marks{marks: map[string]time.Time{}}
}

// Planet tells each kind of fact at a later time than the one before: a fact at or before the latest heard was heard.
type Marks struct {
	handling sync.Mutex

	mu    sync.Mutex
	marks map[string]time.Time
}

func (m *Marks) once(kind string, at time.Time, tell func() error) error {
	m.handling.Lock()
	defer m.handling.Unlock()

	if !at.After(m.mark(kind)) {
		return nil
	}
	if err := tell(); err != nil {
		return err
	}

	m.mu.Lock()
	m.marks[kind] = at
	m.mu.Unlock()
	return nil
}

func (m *Marks) mark(kind string) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.marks[kind]
}

type savedMark struct {
	Kind string
	At   int64
}

func (m *Marks) Name() string { return "heard" }

func (m *Marks) Hold() func() {
	m.handling.Lock()
	return m.handling.Unlock
}

func (m *Marks) Save() ([]byte, error) {
	m.mu.Lock()
	saved := make([]savedMark, 0, len(m.marks))
	for kind, at := range m.marks {
		saved = append(saved, savedMark{Kind: kind, At: evidence.Nanos(at)})
	}
	m.mu.Unlock()

	return evidence.Encode(saved)
}

func (m *Marks) Load(data []byte) error {
	var saved []savedMark
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	marks := make(map[string]time.Time, len(saved))
	for _, mark := range saved {
		marks[mark.Kind] = evidence.Time(mark.At)
	}

	m.mu.Lock()
	m.marks = marks
	m.mu.Unlock()
	return nil
}

func (m *Marks) Forget(before time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for kind, at := range m.marks {
		if at.Before(before) {
			delete(m.marks, kind)
		}
	}
}
