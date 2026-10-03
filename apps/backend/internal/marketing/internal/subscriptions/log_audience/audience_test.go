package log_audience_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/log_audience"
)

func TestEveryCallIsLoggedAndSucceeds(t *testing.T) {
	var logs bytes.Buffer
	a := log_audience.New(slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, a.Join(t.Context(), "ada@example.com"))
	require.NoError(t, a.Invite(t.Context(), "ada@example.com"))
	require.NoError(t, a.Leave(t.Context(), "ada@example.com"))
	require.NoError(t, a.Forget(t.Context(), "ada@example.com"))

	assert.Contains(t, logs.String(), "level=WARN")
	for _, call := range []string{"call=join", "call=invite", "call=leave", "call=forget"} {
		assert.Contains(t, logs.String(), call)
	}
	assert.Contains(t, logs.String(), "address=ada@example.com")
}

func TestAnInvitationIsConfirmedAtOnce(t *testing.T) {
	joined, err := log_audience.New(slog.New(slog.DiscardHandler)).Joined(t.Context(), "ada@example.com")

	require.NoError(t, err)
	assert.True(t, joined, "nobody can click a confirmation that was never sent")
}
