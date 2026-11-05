package inprocess_feed

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Reader answers what a key shows now; false while it shows nothing yet.
type Reader[K comparable, V proto.Message] interface {
	Live(ctx context.Context, key K) (V, bool, error)
}

const (
	tick  = 250 * time.Millisecond
	reads = 8
)

func New[K comparable, V proto.Message](name string, reader Reader[K, V], clock cptime.Clock, every time.Duration) *Feed[K, V] {
	return &Feed[K, V]{
		name:   name,
		reader: reader,
		clock:  clock,
		every:  every,
		wake:   make(chan struct{}, 1),
		views:  map[K]*view[V]{},
	}
}

// Feed reads each key its streams follow once per period, whatever the number of streams, and sends what changed.
type Feed[K comparable, V proto.Message] struct {
	name   string
	reader Reader[K, V]
	clock  cptime.Clock
	every  time.Duration
	wake   chan struct{}

	mu    sync.Mutex
	views map[K]*view[V]
}

type view[V proto.Message] struct {
	streams *cpcolls.Set[chan V]
	shown   V
	shows   bool
	readAt  time.Time
}

func (v *view[V]) due(now time.Time, every time.Duration) bool {
	return v.readAt.IsZero() || now.Sub(v.readAt) >= every
}

func (f *Feed[K, V]) Subscribe(ctx context.Context, key K) <-chan V {
	values := make(chan V, 1)

	f.mu.Lock()
	shown, followed := f.views[key]
	if !followed {
		shown = &view[V]{streams: cpcolls.NewSet[chan V]()}
		f.views[key] = shown
	}
	shown.streams.Add(values)
	if shown.shows {
		values <- shown.shown
	}
	f.mu.Unlock()

	if !followed {
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		shown.streams.Delete(values)
		if shown.streams.Empty() {
			delete(f.views, key)
		}
		close(values)
	}()

	return values
}

func (f *Feed[K, V]) Name() string { return f.name }

func (f *Feed[K, V]) Run(ctx context.Context) {
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

func (f *Feed[K, V]) Refresh(ctx context.Context) {
	group := errgroup.Group{}
	group.SetLimit(reads)
	for _, key := range f.due() {
		group.Go(func() error {
			f.read(ctx, key)
			return nil
		})
	}
	_ = group.Wait()
}

func (f *Feed[K, V]) due() []K {
	now := f.clock.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	var due []K
	for key, shown := range f.views {
		if shown.due(now, f.every) {
			shown.readAt = now
			due = append(due, key)
		}
	}
	return due
}

func (f *Feed[K, V]) read(ctx context.Context, key K) {
	value, shows, err := f.reader.Live(ctx, key)
	if err != nil || !shows {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	shown, followed := f.views[key]
	if !followed || (shown.shows && proto.Equal(shown.shown, value)) {
		return
	}
	shown.shown, shown.shows = value, true
	shown.streams.ForEach(func(stream chan V) {
		select {
		case <-stream:
		default:
		}
		stream <- value
	})
}
