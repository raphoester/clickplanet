package inmemory_tile_storage

import (
	"context"
	"log/slog"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The open streams: each gets every change, and one that falls a buffer behind is cut off.
type feed struct {
	mu          sync.Mutex
	subscribers *cpcolls.Set[chan clicks.Change]
	buffer      int
	logger      *slog.Logger
}

func newFeed(buffer int, logger *slog.Logger) *feed {
	return &feed{subscribers: cpcolls.NewSet[chan clicks.Change](), buffer: buffer, logger: logger}
}

func (f *feed) subscribe(ctx context.Context) <-chan clicks.Change {
	changes := make(chan clicks.Change, f.buffer)

	f.mu.Lock()
	f.subscribers.Add(changes)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		f.unsubscribe(changes)
	}()

	return changes
}

func (f *feed) publish(change clicks.Change) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var behind []chan clicks.Change
	f.subscribers.ForEach(func(changes chan clicks.Change) {
		select {
		case changes <- change:
		default:
			behind = append(behind, changes)
		}
	})
	for _, changes := range behind {
		f.logger.Warn("cut off a map subscriber that fell behind", slog.Int("buffer", f.buffer))
		f.unsubscribe(changes)
	}
}

func (f *feed) publishUpdates(updates []clicks.TileUpdate) {
	for i := range updates {
		f.publish(clicks.Change{Update: &updates[i]})
	}
}

func (f *feed) unsubscribe(changes chan clicks.Change) {
	if !f.subscribers.Contains(changes) {
		return
	}

	f.subscribers.Delete(changes)
	close(changes)
}
