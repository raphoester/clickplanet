package inprocess_level_feed

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// A take is one gain, and a player clicks fast: a stream this far behind drops what comes next.
const subscriberBuffer = 32

// Feed tells an account's open streams, in this process, each level it reaches and the XP each fact counted gave it. It
// keeps what each source gave the account while one of its streams is open, and only then: the XP gained is the
// difference.
type Feed struct {
	mu          sync.Mutex
	subscribers map[accounts.AccountID]*cpcolls.Set[chan levels.Told]
	earned      map[accounts.AccountID]levels.Earned
}

func New() *Feed {
	return &Feed{
		subscribers: map[accounts.AccountID]*cpcolls.Set[chan levels.Told]{},
		earned:      map[accounts.AccountID]levels.Earned{},
	}
}

// Subscribe follows the account from what its stats gave as the stream opened: earned, none for an account with no
// stats yet. A second stream of the account keeps what the first one knows, if it is more.
func (f *Feed) Subscribe(ctx context.Context, account accounts.AccountID, earned levels.Earned) <-chan levels.Told {
	told := make(chan levels.Told, subscriberBuffer)

	f.mu.Lock()
	if f.subscribers[account] == nil {
		f.subscribers[account] = cpcolls.NewSet[chan levels.Told]()
	}
	f.subscribers[account].Add(told)
	f.earned[account] = earned.Most(f.earned[account])
	f.mu.Unlock()

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()
		f.subscribers[account].Delete(told)
		if f.subscribers[account].Empty() {
			delete(f.subscribers, account)
			delete(f.earned, account)
		}
		close(told)
	}()

	return told
}

func (f *Feed) Publish(account accounts.AccountID, reached levels.Standing) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.tell(account, levels.Reached{Standing: reached})
}

// Count tells what the stats give the account now past what it was told before, with its level on its bar. Nothing is
// told when no stream of the account is open, or when no source gives more: a fact told twice, or read late.
func (f *Feed) Count(account accounts.AccountID, earned levels.Earned, standing levels.Standing) {
	f.mu.Lock()
	defer f.mu.Unlock()

	known, open := f.earned[account]
	if !open {
		return
	}
	f.earned[account] = known.Most(earned)
	if gains := earned.Since(known); len(gains) > 0 {
		f.tell(account, levels.Gained{Gains: gains, Standing: standing})
	}
}

func (f *Feed) tell(account accounts.AccountID, told levels.Told) {
	f.subscribers[account].ForEach(func(subscriber chan levels.Told) {
		select {
		case subscriber <- told:
		default:
		}
	})
}
