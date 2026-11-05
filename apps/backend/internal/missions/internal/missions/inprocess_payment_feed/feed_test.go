package inprocess_payment_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inprocess_payment_feed"
)

var (
	ada = missions.AccountID{15: 1}
	bob = missions.AccountID{15: 2}
)

func told(paid <-chan struct{}) bool {
	select {
	case <-paid:
		return true
	default:
		return false
	}
}

func TestEveryStreamOfTheAccountIsToldAndNoOtherAccountIs(t *testing.T) {
	feed := inprocess_payment_feed.New()
	first, second := feed.Subscribe(t.Context(), ada), feed.Subscribe(t.Context(), ada)
	other := feed.Subscribe(t.Context(), bob)

	feed.Publish(ada)

	assert.True(t, told(first))
	assert.True(t, told(second))
	assert.False(t, told(other))
}

func TestPaymentsToldBeforeTheStreamReadsAreOneRead(t *testing.T) {
	feed := inprocess_payment_feed.New()
	paid := feed.Subscribe(t.Context(), ada)

	for range 5 {
		feed.Publish(ada)
	}

	assert.True(t, told(paid))
	assert.False(t, told(paid))
}

func TestAClosedStreamIsForgottenAndItsChannelClosed(t *testing.T) {
	feed := inprocess_payment_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	paid := feed.Subscribe(ctx, ada)

	cancel()

	require.Eventually(t, func() bool {
		select {
		case _, open := <-paid:
			return !open
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	feed.Publish(ada)
}

func TestPublishingToAnAccountWithNoStreamDoesNothing(t *testing.T) {
	assert.NotPanics(t, func() { inprocess_payment_feed.New().Publish(ada) })
}
