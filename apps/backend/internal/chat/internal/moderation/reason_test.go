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

func TestAReasonIsCountedInRunesNotBytes(t *testing.T) {
	_, err := moderation.ReasonOf(strings.Repeat("🌍", moderation.MaxReasonLength))

	require.NoError(t, err)
}

func TestAReasonKeepsItsLinesAndLosesEveryOtherControlCharacter(t *testing.T) {
	reason, err := moderation.ReasonOf("  a slur\r\nsecond\toffence\rand a bell\x07 and a NUL\x00  ")

	require.NoError(t, err)
	assert.Equal(t, moderation.Reason("a slur\nsecond offence\nand a bell and a NUL"), reason)
}
