package log_mailer_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/log_mailer"
)

func TestALetterIsLoggedWithItsCode(t *testing.T) {
	var logs bytes.Buffer
	m := log_mailer.New(slog.New(slog.NewTextHandler(&logs, nil)))

	require.NoError(t, m.Send(t.Context(), "player@example.com", signin.CodeLetter("123456")))

	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "to=player@example.com")
	assert.Contains(t, logs.String(), `subject="Your ClickPlanet code: 123456"`)
}
