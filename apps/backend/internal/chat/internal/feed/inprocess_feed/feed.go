// Package inprocess_feed fans every update out to each open stream, in this process. It keeps nothing: a client
// that was not listening reads the history instead.
package inprocess_feed

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const defaultSubscriberBuffer = 256

// New takes the updates one stream may fall behind by before it misses some; zero is the default.
func New(subscriberBuffer int, logger *slog.Logger) *Feed {
	if subscriberBuffer <= 0 {
		subscriberBuffer = defaultSubscriberBuffer
	}

	return &Feed{
		buffer:      subscriberBuffer,
		logger:      logger,
		subscribers: cpcolls.NewSet[*subscriber](),
	}
}

type Feed struct {
	buffer int
	logger *slog.Logger

	mu          sync.Mutex
	subscribers *cpcolls.Set[*subscriber]
}

type subscriber struct {
	ch      chan feed.Update
	dropped atomic.Uint64
}

// Subscribe is one stream's updates, until ctx ends: then the channel closes.
func (f *Feed) Subscribe(ctx context.Context) (<-chan feed.Update, error) {
	sub := &subscriber{ch: make(chan feed.Update, f.buffer)}

	f.mu.Lock()
	f.subscribers.Add(sub)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		f.subscribers.Delete(sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

const dropLogInterval = 100

// Publish never blocks: a stream too slow to keep up misses the update.
func (f *Feed) Publish(update feed.Update) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers.ForEach(func(sub *subscriber) {
		select {
		case sub.ch <- update:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				f.logger.Warn("dropped a chat update for a slow subscriber",
					slog.String("messageId", string(update.MessageID())),
					slog.Uint64("droppedTotal", dropped),
				)
			}
		}
	})
}

// Dropped is how many updates the open streams missed.
func (f *Feed) Dropped() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	var total uint64
	f.subscribers.ForEach(func(sub *subscriber) {
		total += sub.dropped.Load()
	})
	return total
}
