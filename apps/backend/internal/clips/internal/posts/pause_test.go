package posts_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAPauseEndsAfterItsTime(t *testing.T) {
	require.NoError(t, posts.SystemPause{}.Wait(t.Context(), time.Millisecond))
}

func TestAPauseEndsAtOnceWithItsContext(t *testing.T) {
	stopped, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, posts.SystemPause{}.Wait(stopped, time.Hour), context.Canceled)
}
