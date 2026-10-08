package inprocess_race_feed

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Reader interface {
	Race(ctx context.Context) (*seasonsv1.Race, error)
}

const (
	Every   = time.Second
	AtLeast = time.Minute

	tick = 250 * time.Millisecond
)

func New(reader Reader, clock cptime.Clock) *Feed {
	return &Feed{
		reader:  reader,
		clock:   clock,
		wake:    make(chan struct{}, 1),
		streams: cpcolls.NewSet[chan *seasonsv1.Race](),
	}
}

type Feed struct {
	reader Reader
	clock  cptime.Clock
	wake   chan struct{}

	mu      sync.Mutex
	streams *cpcolls.Set[chan *seasonsv1.Race]
	race    *seasonsv1.Race
	readAt  time.Time
	moved   bool
}

func (f *Feed) Subscribe(ctx context.Context) <-chan *seasonsv1.Race {
	races := make(chan *seasonsv1.Race, 1)

	f.mu.Lock()
	first := f.streams.Empty()
	f.streams.Add(races)
	if f.race != nil {
		races <- f.race
	}
	f.mu.Unlock()

	if first {
		f.nudge()
	}

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		f.streams.Delete(races)
		if f.streams.Empty() {
			f.race, f.readAt, f.moved = nil, time.Time{}, false
		}
		close(races)
	}()

	return races
}

func (f *Feed) MarkCounted() {
	f.mu.Lock()
	f.moved = true
	f.mu.Unlock()

	f.nudge()
}

func (f *Feed) nudge() {
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

func (f *Feed) Name() string { return "seasons-race" }

func (f *Feed) Run(ctx context.Context) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-f.wake:
		}
		f.Refresh(ctx)
	}
}

func (f *Feed) Refresh(ctx context.Context) {
	if !f.due() {
		return
	}
	race, err := f.reader.Race(ctx)

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.streams.Empty() {
		return
	}
	if err != nil {
		f.moved = true
		return
	}
	if f.race != nil && proto.Equal(f.race, race) {
		return
	}
	f.race = race
	f.streams.ForEach(func(stream chan *seasonsv1.Race) {
		select {
		case <-stream:
		default:
		}
		stream <- race
	})
}

func (f *Feed) due() bool {
	now := f.clock.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.streams.Empty() || !f.stale(now) {
		return false
	}
	f.moved, f.readAt = false, now
	return true
}

func (f *Feed) stale(now time.Time) bool {
	elapsed := now.Sub(f.readAt)
	return f.readAt.IsZero() || elapsed >= AtLeast || (f.moved && elapsed >= Every)
}
