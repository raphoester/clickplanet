package inprocess_title_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inprocess_title_feed"
)

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func received(t *testing.T, earned <-chan titles.IDs) titles.IDs {
	t.Helper()

	select {
	case ids := <-earned:
		return ids
	case <-time.After(time.Second):
		require.Fail(t, "nothing was received")
		return nil
	}
}

func TestEveryStreamOfTheAccountGetsItsTitlesAndNoOtherAccountDoes(t *testing.T) {
	feed := inprocess_title_feed.New()
	first, second := feed.Subscribe(t.Context(), ada), feed.Subscribe(t.Context(), ada)
	other := feed.Subscribe(t.Context(), bob)

	feed.Publish(ada, titles.IDs{"warlord"})

	assert.Equal(t, titles.IDs{"warlord"}, received(t, first))
	assert.Equal(t, titles.IDs{"warlord"}, received(t, second))
	select {
	case ids := <-other:
		assert.Fail(t, "another account received titles", "%v", ids)
	default:
	}
}

func TestAClosedStreamIsForgottenAndItsChannelClosed(t *testing.T) {
	feed := inprocess_title_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	earned := feed.Subscribe(ctx, ada)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-earned:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	feed.Publish(ada, titles.IDs{"warlord"})
}

func TestAFullStreamDropsRatherThanBlocks(t *testing.T) {
	feed := inprocess_title_feed.New()
	earned := feed.Subscribe(t.Context(), ada)

	for range 20 {
		feed.Publish(ada, titles.IDs{"settler"})
	}

	assert.Len(t, earned, 8)
}

func TestPublishingToAnAccountWithNoStreamDoesNothing(t *testing.T) {
	assert.NotPanics(t, func() { inprocess_title_feed.New().Publish(ada, titles.IDs{"warlord"}) })
}
