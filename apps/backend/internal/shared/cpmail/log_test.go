package cpmail_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

func TestAMailIsLoggedWithWhatItSaysAndAnIDOfItsOwn(t *testing.T) {
	var logs bytes.Buffer
	m := cpmail.NewLog(slog.New(slog.NewTextHandler(&logs, nil)))

	first, err := m.Send(t.Context(), mail)
	require.NoError(t, err)
	second, err := m.Send(t.Context(), mail)
	require.NoError(t, err)

	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "to=player@example.com")
	assert.Contains(t, logs.String(), `subject="Re: your idea"`)
	assert.Contains(t, logs.String(), "text=Thanks!")
	assert.NotEqual(t, first, second)
	assert.Contains(t, string(first), "@example.com")
}
