package inprocess_level_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inprocess_level_feed"
)

var (
	ada = accounts.AccountID{15: 1}
	bob = accounts.AccountID{15: 2}

	fifth = levels.Curve{}.StandingOf(900)
)

func received(t *testing.T, reached <-chan levels.Standing) levels.Standing {
	t.Helper()

	select {
	case standing := <-reached:
		return standing
	case <-time.After(time.Second):
		require.Fail(t, "nothing was received")
		return levels.Standing{}
	}
}

func TestEveryStreamOfTheAccountGetsItsLevelAndNoOtherAccountDoes(t *testing.T) {
	feed := inprocess_level_feed.New()
	first, second := feed.Subscribe(t.Context(), ada), feed.Subscribe(t.Context(), ada)
	other := feed.Subscribe(t.Context(), bob)

	feed.Publish(ada, fifth)

	assert.Equal(t, fifth, received(t, first))
	assert.Equal(t, fifth, received(t, second))
	select {
	case standing := <-other:
		assert.Fail(t, "another account received a level", "%v", standing)
	default:
	}
}

func TestAClosedStreamIsForgottenAndItsChannelClosed(t *testing.T) {
	feed := inprocess_level_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	reached := feed.Subscribe(ctx, ada)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-reached:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	feed.Publish(ada, fifth)
}

func TestAFullStreamDropsRatherThanBlocks(t *testing.T) {
	feed := inprocess_level_feed.New()
	reached := feed.Subscribe(t.Context(), ada)

	for range 20 {
		feed.Publish(ada, fifth)
	}

	assert.Len(t, reached, 8)
}

func TestPublishingToAnAccountWithNoStreamDoesNothing(t *testing.T) {
	assert.NotPanics(t, func() { inprocess_level_feed.New().Publish(ada, fifth) })
}
