package moderation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

func TestNoReasonIsAReasonToo(t *testing.T) {
	for _, value := range []string{"", " \n\t "} {
		reason, err := moderation.ReasonOf(value)

		require.NoError(t, err, "%q", value)
		assert.Equal(t, moderation.NoReason, reason, "%q", value)
	}
}

func TestAnOverlongReasonIsRefused(t *testing.T) {
	_, err := moderation.ReasonOf(strings.Repeat("a", moderation.MaxReasonLength+1))

	require.ErrorIs(t, err, moderation.ErrInvalidReason)
}

func TestAReasonIsCountedInRunesNotBytes(t *testing.T) {
	_, err := moderation.ReasonOf(strings.Repeat("🌍", moderation.MaxReasonLength))

	require.NoError(t, err)
}

func TestAReasonThatIsNotUTF8IsRefused(t *testing.T) {
	_, err := moderation.ReasonOf(string([]byte{0xff, 0xfe, 0xfd}))

	require.ErrorIs(t, err, moderation.ErrInvalidReason)
}

func TestAReasonKeepsItsLinesAndLosesEveryOtherControlCharacter(t *testing.T) {
	reason, err := moderation.ReasonOf("  a slur\r\nsecond\toffence\rand a bell\x07 and a NUL\x00  ")

	require.NoError(t, err)
	assert.Equal(t, moderation.Reason("a slur\nsecond offence\nand a bell and a NUL"), reason)
}
