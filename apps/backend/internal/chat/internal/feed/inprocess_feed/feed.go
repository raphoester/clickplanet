package inprocess_feed

import (
	"context"
	"log/slog"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const defaultSubscriberBuffer = 256

func New(subscriberBuffer int, logger *slog.Logger) *Feed {
	if subscriberBuffer <= 0 {
		subscriberBuffer = defaultSubscriberBuffer
	}

	return &Feed{
		buffer:      subscriberBuffer,
		logger:      logger,
		subscribers: cpcolls.NewSet[chan feed.Update](),
	}
}

type Feed struct {
	buffer int
	logger *slog.Logger

	mu          sync.Mutex
	subscribers *cpcolls.Set[chan feed.Update]
}

func (f *Feed) Subscribe(ctx context.Context) (<-chan feed.Update, error) {
	updates := make(chan feed.Update, f.buffer)

	f.mu.Lock()
	f.subscribers.Add(updates)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		f.unsubscribe(updates)
	}()

	return updates, nil
}

func (f *Feed) Publish(update feed.Update) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var behind []chan feed.Update
	f.subscribers.ForEach(func(updates chan feed.Update) {
		select {
		case updates <- update:
		default:
			behind = append(behind, updates)
		}
	})
	for _, updates := range behind {
		f.logger.Warn("cut off a chat subscriber that fell behind", slog.Int("buffer", f.buffer))
		f.unsubscribe(updates)
	}
}

func (f *Feed) unsubscribe(updates chan feed.Update) {
	if !f.subscribers.Contains(updates) {
		return
	}

	f.subscribers.Delete(updates)
	close(updates)
}
