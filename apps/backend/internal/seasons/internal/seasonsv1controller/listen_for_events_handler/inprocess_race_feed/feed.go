package inprocess_race_feed

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Reader interface {
	Race(ctx context.Context) (*seasonsv1.Race, error)
}

const Every = 10 * time.Second

func New(reader Reader) *Feed {
	return &Feed{
		reader:  reader,
		wake:    make(chan struct{}, 1),
		streams: cpcolls.NewSet[chan *seasonsv1.Race](),
	}
}

type Feed struct {
	reader Reader
	wake   chan struct{}

	mu      sync.Mutex
	streams *cpcolls.Set[chan *seasonsv1.Race]
	race    *seasonsv1.Race
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
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		f.streams.Delete(races)
		if f.streams.Empty() {
			f.race = nil
		}
		close(races)
	}()

	return races
}

func (f *Feed) Name() string { return "seasons-race" }

func (f *Feed) Run(ctx context.Context) {
	ticker := time.NewTicker(Every)
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
	if !f.followed() {
		return
	}
	race, err := f.reader.Race(ctx)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.streams.Empty() || (f.race != nil && proto.Equal(f.race, race)) {
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

func (f *Feed) followed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return !f.streams.Empty()
}
