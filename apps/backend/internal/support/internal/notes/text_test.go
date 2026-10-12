package notes_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func TestEmptyTextIsRefused(t *testing.T) {
	_, err := notes.TextOf(" \n\t ")

	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestOverlongTextIsRefused(t *testing.T) {
	_, err := notes.TextOf(strings.Repeat("a", notes.MaxTextLength+1))

	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestLengthIsCountedInRunesNotBytes(t *testing.T) {
	_, err := notes.TextOf(strings.Repeat("🌍", notes.MaxTextLength))

	require.NoError(t, err)
}

func TestInvalidUTF8IsRefused(t *testing.T) {
	_, err := notes.TextOf(string([]byte{0xff, 0xfe, 0xfd}))

	require.ErrorIs(t, err, notes.ErrInvalidText)
}

func TestLinesAreKeptAndEveryOtherControlCharacterIsStripped(t *testing.T) {
	text, err := notes.TextOf("  a map of the day\r\nwith\ttabs\rand a bell\x07 and a NUL\x00  ")

	require.NoError(t, err)
	assert.Equal(t, "a map of the day\nwith tabs\nand a bell and a NUL", text)
}
