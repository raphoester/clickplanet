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
	Turn(ctx context.Context) (*gamemasterv1.Turn, error)
}

// The schedule runner ticks every second too: a change of turn is seen within one.
const Every = time.Second

func New(reader Reader) *Feed {
	return &Feed{
		reader:  reader,
		wake:    make(chan struct{}, 1),
		streams: cpcolls.NewSet[chan *gamemasterv1.Turn](),
	}
}

// Feed reads the turn while anyone follows it, and hands each change to every stream.
type Feed struct {
	reader Reader
	wake   chan struct{}

	mu      sync.Mutex
	streams *cpcolls.Set[chan *gamemasterv1.Turn]
	turn    *gamemasterv1.Turn
}

func (f *Feed) Subscribe(ctx context.Context) <-chan *gamemasterv1.Turn {
	turns := make(chan *gamemasterv1.Turn, 1)

	f.mu.Lock()
	first := f.streams.Empty()
	f.streams.Add(turns)
	if f.turn != nil {
		turns <- f.turn
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

		f.streams.Delete(turns)
		if f.streams.Empty() {
			f.turn = nil
		}
		close(turns)
	}()

	return turns
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
	turn, err := f.reader.Turn(ctx)
	if err != nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.streams.Empty() || (f.turn != nil && proto.Equal(f.turn, turn)) {
		return
	}
	f.turn = turn
	f.streams.ForEach(func(stream chan *gamemasterv1.Turn) {
		select {
		case <-stream:
		default:
		}
		stream <- turn
	})
}

func (f *Feed) followed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return !f.streams.Empty()
}
