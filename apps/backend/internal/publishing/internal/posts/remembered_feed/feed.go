package remembered_feed

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
)

type Feeds interface {
	Posted(ctx context.Context, since time.Time) (calendar.Feed, error)
}

func New(inner Feeds) *Feed {
	return &Feed{inner: inner}
}

// What the service lists, and the posts this server made until the list shows them: a list read right after a post
// may not hold it yet, and a post missing from the feed would let the next one crowd it. A post is forgotten once the
// list shows it, or once it is a day old, out of every budget.
type Feed struct {
	inner Feeds
	mu    sync.Mutex
	made  calendar.Feed
}

func (f *Feed) Posted(ctx context.Context, since time.Time) (calendar.Feed, error) {
	listed, err := f.inner.Posted(ctx, since)
	if err != nil {
		return nil, err //nolint:wrapcheck // a decorator adds what it remembers, not a sentence.
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.made = slices.DeleteFunc(f.made, func(made calendar.Posting) bool {
		return made.At.Before(since) || slices.ContainsFunc(listed, func(posting calendar.Posting) bool { return posting.ID == made.ID })
	})
	return append(slices.Clone(listed), f.made...), nil
}

// A post this server made, on the platforms that took it.
func (f *Feed) Remember(made calendar.Posting) {
	if made.ID == "" || len(made.Platforms) == 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.made = append(f.made, calendar.Posting{ID: made.ID, At: made.At, Platforms: slices.Clone(made.Platforms)})
}
