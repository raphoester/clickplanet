package inprocess_level_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const subscriberBuffer = 8

type Feed struct {
	mu          sync.Mutex
	subscribers map[accounts.AccountID]*cpcolls.Set[chan levels.Standing]
}

func New() *Feed {
	return &Feed{subscribers: map[accounts.AccountID]*cpcolls.Set[chan levels.Standing]{}}
}

func (f *Feed) Subscribe(ctx context.Context, account accounts.AccountID) <-chan levels.Standing {
	reached := make(chan levels.Standing, subscriberBuffer)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan levels.Standing]()
	}
	f.subscribers[account].Add(reached)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(reached)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
		}
		close(reached)
	}()

	return reached
}

func (f *Feed) Publish(account accounts.AccountID, reached levels.Standing) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers[account].ForEach(func(subscriber chan levels.Standing) {
		select {
		case subscriber <- reached:
		default:
		}
	})
}
