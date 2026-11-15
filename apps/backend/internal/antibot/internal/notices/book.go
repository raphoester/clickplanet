package notices

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// A notice tells an account that another one played from its connection at the same time, and that one player plays
// one account. Unlike a warning, nothing follows from it: no ban reads it.
type Notice struct {
	Account        string
	Alongside      string
	At             time.Time
	AcknowledgedAt time.Time
}

func (n Notice) Pending() bool { return n.AcknowledgedAt.IsZero() }

type Store interface {
	// Noticed answers the account's notice, if it was ever noticed.
	Noticed(ctx context.Context, account string) (Notice, bool, error)
	// Keep keeps a notice unless its account was noticed before, and answers whether it kept it.
	Keep(ctx context.Context, notice Notice) (bool, error)
	// Acknowledge keeps the first acknowledgement of the account's notice; an account never noticed changes nothing.
	Acknowledge(ctx context.Context, account string, at time.Time) error
}

// Whoever shows the account its notice while it plays.
type Feed interface {
	Publish(notice Notice)
}

func NewBook(clock cptime.Clock, store Store, feed Feed) *Book {
	return &Book{clock: clock, store: store, feed: feed}
}

type Book struct {
	clock cptime.Clock
	store Store
	feed  Feed
}

// Each account is noticed once, ever: false kept none, as the account was noticed before.
func (b *Book) Notify(ctx context.Context, account, alongside string) (Notice, bool, error) {
	notice := Notice{Account: account, Alongside: alongside, At: b.clock.Now()}

	kept, err := b.store.Keep(ctx, notice)
	if err != nil {
		return Notice{}, false, fmt.Errorf("failed to notice %q: %w", account, err)
	}
	if !kept {
		return Notice{}, false, nil
	}
	b.feed.Publish(notice)
	return notice, true, nil
}

func (b *Book) Acknowledge(ctx context.Context, account string) error {
	if err := b.store.Acknowledge(ctx, account, b.clock.Now()); err != nil {
		return fmt.Errorf("failed to keep the notice of %q acknowledged: %w", account, err)
	}
	return nil
}

func (b *Book) Pending(ctx context.Context, account string) (bool, error) {
	notice, noticed, err := b.Notice(ctx, account)
	return noticed && notice.Pending(), err
}

func (b *Book) Notice(ctx context.Context, account string) (Notice, bool, error) {
	if account == "" {
		return Notice{}, false, nil
	}

	notice, noticed, err := b.store.Noticed(ctx, account)
	if err != nil {
		return Notice{}, false, fmt.Errorf("failed to read the notice of %q: %w", account, err)
	}
	return notice, noticed, nil
}
