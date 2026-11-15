package inprocess_notice_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const subscriberBuffer = 1

var _ notices.Feed = (*Feed)(nil)

type Feed struct {
	mu          sync.Mutex
	subscribers map[string]*cpcolls.Set[chan notices.Notice]
}

func New() *Feed {
	return &Feed{subscribers: map[string]*cpcolls.Set[chan notices.Notice]{}}
}

func (f *Feed) Subscribe(ctx context.Context, account string) <-chan notices.Notice {
	given := make(chan notices.Notice, subscriberBuffer)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan notices.Notice]()
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

func (f *Feed) Publish(notice notices.Notice) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.subscribers[notice.Account].ForEach(func(subscriber chan notices.Notice) {
		select {
		case subscriber <- notice:
		default:
		}
	})
}
