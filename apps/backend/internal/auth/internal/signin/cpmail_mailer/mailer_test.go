package cpmail_mailer_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/cpmail_mailer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

func TestTheLetterGoesFromTheSenderWithItsCode(t *testing.T) {
	var logs bytes.Buffer
	m := cpmail_mailer.New(cpmail.NewLog(slog.New(slog.NewTextHandler(&logs, nil))),
		cpmail.Named{Address: "noreply@example.com", Name: "ClickPlanet"})

	require.NoError(t, m.Send(t.Context(), "player@example.com", signin.CodeLetter("123456")))

	assert.Contains(t, logs.String(), "to=player@example.com")
	assert.Contains(t, logs.String(), `subject="Your ClickPlanet code: 123456"`)
	assert.Contains(t, logs.String(), "@example.com>")
}
