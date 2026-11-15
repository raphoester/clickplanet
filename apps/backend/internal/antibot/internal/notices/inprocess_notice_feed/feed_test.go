package inprocess_notice_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices/inprocess_notice_feed"
)

var at = time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)

func TestANoticeReachesEveryStreamOfItsAccountAndNoOther(t *testing.T) {
	feed := inprocess_notice_feed.New()
	phone := feed.Subscribe(t.Context(), "ada")
	desktop := feed.Subscribe(t.Context(), "ada")
	other := feed.Subscribe(t.Context(), "grace")

	feed.Publish(notices.Notice{Account: "ada", Alongside: "grace", At: at})

	want := notices.Notice{Account: "ada", Alongside: "grace", At: at}
	assert.Equal(t, want, <-phone)
	assert.Equal(t, want, <-desktop)
	assert.Empty(t, other)
}

func TestAStreamThatEndsStopsListening(t *testing.T) {
	feed := inprocess_notice_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	given := feed.Subscribe(ctx, "ada")

	cancel()

	_, open := <-given
	require.False(t, open)
	feed.Publish(notices.Notice{Account: "ada", At: at})
}
