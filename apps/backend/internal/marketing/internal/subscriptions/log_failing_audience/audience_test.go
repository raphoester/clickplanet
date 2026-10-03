package log_failing_audience_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/log_failing_audience"
)

func TestAFailedCallIsLoggedWithoutTheAddressAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	inner := &subscriptions.FakeAudience{}
	inner.Fail("Join", "Joined")
	a := log_failing_audience.New(inner, slog.New(slog.NewTextHandler(&logs, nil)))

	require.ErrorIs(t, a.Join(t.Context(), "ada@example.com"), subscriptions.ErrFakeAudienceDown)
	_, err := a.Joined(t.Context(), "ada@example.com")
	require.ErrorIs(t, err, subscriptions.ErrFakeAudienceDown)

	assert.Contains(t, logs.String(), "level=ERROR")
	assert.Contains(t, logs.String(), "call=join")
	assert.Contains(t, logs.String(), "call=joined")
	assert.NotContains(t, logs.String(), "ada@example.com")
}

func TestASuccessfulCallLogsNothing(t *testing.T) {
	var logs bytes.Buffer
	inner := &subscriptions.FakeAudience{}
	inner.Confirm("ada@example.com")
	a := log_failing_audience.New(inner, slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, a.Invite(t.Context(), "ada@example.com"))
	require.NoError(t, a.Leave(t.Context(), "ada@example.com"))
	require.NoError(t, a.Forget(t.Context(), "ada@example.com"))
	joined, err := a.Joined(t.Context(), "ada@example.com")
	require.NoError(t, err)

	assert.True(t, joined)
	assert.Empty(t, logs.String())
	assert.Len(t, inner.Calls(), 4)
}
