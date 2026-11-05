package inprocess_title_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const subscriberBuffer = 8

type Feed struct {
	mu          sync.Mutex
	subscribers map[accounts.AccountID]*cpcolls.Set[chan titles.IDs]
}

func New() *Feed {
	return &Feed{subscribers: map[accounts.AccountID]*cpcolls.Set[chan titles.IDs]{}}
}

func (f *Feed) Subscribe(ctx context.Context, account accounts.AccountID) <-chan titles.IDs {
	earned := make(chan titles.IDs, subscriberBuffer)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan titles.IDs]()
	}
	f.subscribers[account].Add(earned)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(earned)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
		}
		close(earned)
	}()

	return earned
}

func (f *Feed) Publish(account accounts.AccountID, earned titles.IDs) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers[account].ForEach(func(subscriber chan titles.IDs) {
		select {
		case subscriber <- earned:
		default:
		}
	})
}
