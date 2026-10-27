package inprocess_turn_feed

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Reader interface {
	Now(ctx context.Context) (*gamemasterv1.GetTurnResponse, error)
}

// The schedule runner ticks every second too: a change of turn, or of an entry to come, is seen within one.
const Every = time.Second

func New(reader Reader) *Feed {
	return &Feed{
		reader:  reader,
		wake:    make(chan struct{}, 1),
		streams: cpcolls.NewSet[chan *gamemasterv1.GetTurnResponse](),
	}
}

// Feed reads the turn and the timeline while anyone follows them, and hands each change of either to every stream.
type Feed struct {
	reader Reader
	wake   chan struct{}

	mu      sync.Mutex
	streams *cpcolls.Set[chan *gamemasterv1.GetTurnResponse]
	now     *gamemasterv1.GetTurnResponse
}

func (f *Feed) Subscribe(ctx context.Context) <-chan *gamemasterv1.GetTurnResponse {
	nows := make(chan *gamemasterv1.GetTurnResponse, 1)

	f.mu.Lock()
	first := f.streams.Empty()
	f.streams.Add(nows)
	if f.now != nil {
		nows <- f.now
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

		f.streams.Delete(nows)
		if f.streams.Empty() {
			f.now = nil
		}
		close(nows)
	}()

	return nows
}

func (f *Feed) Name() string { return "gamemaster-turn" }

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
	now, err := f.reader.Now(ctx)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.streams.Empty() || (f.now != nil && proto.Equal(f.now, now)) {
		return
	}
	f.now = now
	f.streams.ForEach(func(stream chan *gamemasterv1.GetTurnResponse) {
		select {
		case <-stream:
		default:
		}
		stream <- now
	})
}

func (f *Feed) followed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return !f.streams.Empty()
}
