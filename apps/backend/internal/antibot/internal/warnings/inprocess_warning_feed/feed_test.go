package inprocess_warning_feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings/inprocess_warning_feed"
)

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

func TestAWarningReachesEveryStreamOfItsAccountAndNoOther(t *testing.T) {
	feed := inprocess_warning_feed.New()
	phone := feed.Subscribe(t.Context(), "ada")
	desktop := feed.Subscribe(t.Context(), "ada")
	other := feed.Subscribe(t.Context(), "grace")

	feed.Publish(warnings.Warning{Account: "ada", At: at})

	assert.Equal(t, warnings.Warning{Account: "ada", At: at}, <-phone)
	assert.Equal(t, warnings.Warning{Account: "ada", At: at}, <-desktop)
	assert.Empty(t, other)
}

func TestAStreamThatEndsStopsListening(t *testing.T) {
	feed := inprocess_warning_feed.New()
	ctx, cancel := context.WithCancel(t.Context())
	given := feed.Subscribe(ctx, "ada")

	cancel()

	_, open := <-given
	require.False(t, open)
	feed.Publish(warnings.Warning{Account: "ada", At: at})
}

func TestAWarningToAnAccountWithNoStreamIsDropped(t *testing.T) {
	feed := inprocess_warning_feed.New()

	assert.NotPanics(t, func() { feed.Publish(warnings.Warning{Account: "ada", At: at}) })
}
