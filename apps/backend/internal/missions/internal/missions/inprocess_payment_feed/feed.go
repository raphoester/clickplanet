package inprocess_payment_feed

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

// One slot: payments told while a stream reads the day are one more read, not one each.
func (f *Feed) Subscribe(ctx context.Context, account missions.AccountID) <-chan struct{} {
	paid := make(chan struct{}, 1)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan struct{}]()
	}
	f.subscribers[account].Add(paid)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(paid)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
		}
		close(paid)
	}()

	return paid
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
