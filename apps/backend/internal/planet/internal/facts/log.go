package facts

import (
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Fact func(at time.Time) proto.Message

func NewLog(clock cptime.Clock) *Log {
	return &Log{clock: clock}
}

// Each fact at a later time than the one before, so the time names it and a listener knows one told again.
type Log struct {
	clock cptime.Clock

	mu   sync.Mutex
	last time.Time
	told []proto.Message
}

func (l *Log) Tell(fact Fact) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// The wall clock alone: a fact crosses the process as a timestamp, and the wall can step back.
	at := l.clock.Now().Round(0)
	if !at.After(l.last) {
		at = l.last.Add(time.Nanosecond)
	}
	l.last = at
	l.told = append(l.told, fact(at))
}

func (l *Log) Take() []proto.Message {
	l.mu.Lock()
	defer l.mu.Unlock()

	told := l.told
	l.told = nil
	return told
}
