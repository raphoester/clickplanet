package messages_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

var limits = messages.NewLimits(0)

func TestEmptyTextIsRefused(t *testing.T) {
	_, err := limits.Text("   ")
	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestOverlongTextIsRefused(t *testing.T) {
	_, err := limits.Text(strings.Repeat("a", 281))
	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestLengthIsCountedInRunesNotBytes(t *testing.T) {
	_, err := limits.Text(strings.Repeat("🌍", 280))
	require.NoError(t, err)
}

func TestInvalidUTF8IsRefused(t *testing.T) {
	_, err := limits.Text(string([]byte{0xff, 0xfe, 0xfd}))
	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestControlCharactersAreStripped(t *testing.T) {
	text, err := limits.Text("hello\n{\"ip\":\"forged\"}\r\x00 planet")
	require.NoError(t, err)

	assert.NotContains(t, text, "\n")
	assert.NotContains(t, text, "\r")
	assert.NotContains(t, text, "\x00")
}

func TestTabsBecomeSpaces(t *testing.T) {
	text, err := limits.Text("hello\tplanet")
	require.NoError(t, err)
	assert.Equal(t, "hello planet", text)
}

func TestConfiguredLimitsReplaceTheDefaults(t *testing.T) {
	_, err := messages.NewLimits(5).Text("hello!")
	require.ErrorIs(t, err, messages.ErrInvalidMessage)
}
