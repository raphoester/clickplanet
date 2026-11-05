package inprocess_day_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Feed struct {
	mu          sync.Mutex
	subscribers map[missions.AccountID]*cpcolls.Set[chan struct{}]
}

func New() *Feed {
	return &Feed{subscribers: map[missions.AccountID]*cpcolls.Set[chan struct{}]{}}
}

// One slot: changes told while a stream reads the day are one more read, not one each.
func (f *Feed) Subscribe(ctx context.Context, account missions.AccountID) <-chan struct{} {
	changed := make(chan struct{}, 1)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan struct{}]()
	}
	f.subscribers[account].Add(changed)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(changed)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
		}
		close(changed)
	}()

	return changed
}

func (f *Feed) Publish(account missions.AccountID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers[account].ForEach(func(subscriber chan struct{}) {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	})
}
