package inprocess_day_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inprocess_day_feed"
)

var (
	ada = missions.AccountID{15: 1}
	bob = missions.AccountID{15: 2}
)

func told(changed <-chan struct{}) bool {
	select {
	case <-changed:
		return true
	default:
		return false
	}
}

func TestEveryStreamOfTheAccountIsToldAndNoOtherAccountIs(t *testing.T) {
	feed := inprocess_day_feed.New()
	first, second := feed.Subscribe(t.Context(), ada), feed.Subscribe(t.Context(), ada)
	other := feed.Subscribe(t.Context(), bob)

	feed.Publish(ada)

	assert.True(t, told(first))
	assert.True(t, told(second))
	assert.False(t, told(other))
}

func TestChangesToldBeforeTheStreamReadsAreOneRead(t *testing.T) {
	feed := inprocess_day_feed.New()
	changed := feed.Subscribe(t.Context(), ada)

	for range 5 {
		feed.Publish(ada)
	}

	assert.True(t, told(changed))
	assert.False(t, told(changed))
}

func TestAClosedStreamIsForgottenAndItsChannelClosed(t *testing.T) {
	feed := inprocess_day_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	changed := feed.Subscribe(ctx, ada)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-changed:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	feed.Publish(ada)
}

func TestPublishingToAnAccountWithNoStreamDoesNothing(t *testing.T) {
	assert.NotPanics(t, func() { inprocess_day_feed.New().Publish(ada) })
}
