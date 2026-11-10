package inprocess_warning_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const subscriberBuffer = 1

var _ warnings.Feed = (*Feed)(nil)

type Feed struct {
	mu          sync.Mutex
	subscribers map[string]*cpcolls.Set[chan warnings.Warning]
}

func New() *Feed {
	return &Feed{subscribers: map[string]*cpcolls.Set[chan warnings.Warning]{}}
}

func (f *Feed) Subscribe(ctx context.Context, account string) <-chan warnings.Warning {
	given := make(chan warnings.Warning, subscriberBuffer)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan warnings.Warning]()
	}
	f.subscribers[account].Add(given)
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(given)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
		}
		close(given)
	}()

	return given
}

func (f *Feed) Publish(warning warnings.Warning) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers[warning.Account].ForEach(func(subscriber chan warnings.Warning) {
		select {
		case subscriber <- warning:
		default:
		}
	})
}
