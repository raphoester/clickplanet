package renames_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames"
)

func TestNoReasonIsAReasonToo(t *testing.T) {
	for _, value := range []string{"", " \n\t "} {
		reason, err := renames.ReasonOf(value)

		require.NoError(t, err, "%q", value)
		assert.Equal(t, renames.NoReason, reason, "%q", value)
	}
}

func TestAnOverlongReasonIsRefused(t *testing.T) {
	_, err := renames.ReasonOf(strings.Repeat("a", renames.MaxReasonLength+1))

	require.ErrorIs(t, err, renames.ErrInvalidReason)
}

func TestAReasonIsCountedInRunesNotBytes(t *testing.T) {
	_, err := renames.ReasonOf(strings.Repeat("🌍", renames.MaxReasonLength))

	require.NoError(t, err)
}

func TestAReasonThatIsNotUTF8IsRefused(t *testing.T) {
	_, err := renames.ReasonOf(string([]byte{0xff, 0xfe, 0xfd}))

	require.ErrorIs(t, err, renames.ErrInvalidReason)
}

func TestAReasonKeepsItsLinesAndLosesEveryOtherControlCharacter(t *testing.T) {
	reason, err := renames.ReasonOf("  a slur\r\nsecond\toffence\rand a bell\x07 and a NUL\x00  ")

	require.NoError(t, err)
	assert.Equal(t, renames.Reason("a slur\nsecond offence\nand a bell and a NUL"), reason)
}
