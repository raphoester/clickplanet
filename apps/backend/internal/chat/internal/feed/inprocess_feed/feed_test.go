package inprocess_feed_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/inprocess_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func newFeed(buffer int) *inprocess_feed.Feed {
	return inprocess_feed.New(buffer, slog.New(slog.DiscardHandler))
}

func sent(text string) feed.Update {
	return feed.MessageSent(messages.NewMessage(messages.MessageID(text), time.Time{}, messages.NoAccount, "", text))
}

func TestEverySubscriberHearsEveryUpdateInOrder(t *testing.T) {
	updates := newFeed(0)
	first, err := updates.Subscribe(t.Context())
	require.NoError(t, err)
	second, err := updates.Subscribe(t.Context())
	require.NoError(t, err)

	tally := feed.ReactionsChanged(reactions.TallyFor("hello", nil, 0))
	updates.Publish(sent("hello"))
	updates.Publish(tally)

	for _, heard := range []<-chan feed.Update{first, second} {
		assert.Equal(t, sent("hello"), <-heard)
		assert.Equal(t, tally, <-heard)
	}
}

func TestASubscriptionClosesWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	updates, err := newFeed(0).Subscribe(ctx)
	require.NoError(t, err)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-updates:
			return !open
		default:
			return false
		}
	}, 2*time.Second, time.Millisecond)
}

func TestASlowSubscriberIsClosedRatherThanHoldingThePublisher(t *testing.T) {
	updates := newFeed(1)
	slow, err := updates.Subscribe(t.Context())
	require.NoError(t, err)
	reading, err := updates.Subscribe(t.Context())
	require.NoError(t, err)

	for range 10 {
		updates.Publish(sent("hello"))

		update, open := <-reading
		require.True(t, open, "a subscriber that keeps up is never cut off")
		assert.Equal(t, sent("hello"), update)
	}

	update, open := <-slow
	require.True(t, open)
	assert.Equal(t, sent("hello"), update)
	_, open = <-slow
	assert.False(t, open, "closed, not skipped: its stream ends and the client reads the history again")
}

func TestConcurrentUseIsSafe(t *testing.T) {
	updates := newFeed(1000)
	feedOf, err := updates.Subscribe(t.Context())
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 20 {
				updates.Publish(sent("hello"))
			}
		})
	}
	for range 5 {
		wg.Go(func() {
			ctx, cancel := context.WithCancel(t.Context())
			_, err := updates.Subscribe(ctx)
			cancel()
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	assert.Len(t, feedOf, 200)
}
