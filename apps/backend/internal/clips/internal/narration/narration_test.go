package narration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAScriptIsMeasuredAsItIsSaid(t *testing.T) {
	script := narration.Script{Script: posts.Script{Lines: []string{"Day nine.", "Palau still owns Russia.", "Russia has not commented."}}}

	assert.Equal(t, "Day nine. Palau still owns Russia. Russia has not commented.", script.Said())
	assert.Equal(t, 10, script.Words())
	assert.InDelta(t, 10/3.3, script.Seconds(), 1e-9)
	assert.Empty(t, script.Tells())
}

func TestTheTurnsThatGiveAMachineAwayAreFoundAndSoAreTheSignsAVoiceReadsBadly(t *testing.T) {
	script := narration.Script{Script: posts.Script{Lines: []string{
		"In a stunning turn of events, Palau took 68% of Russia — and only time will tell!", "Buckle up!",
	}}}

	assert.Equal(t, []string{"buckle up", "only time will tell", "turn of events", "a dash", "a digit", "exclamation marks"}, script.Tells())
}

func TestAWriterIsNamedByItsFamilyOrItsID(t *testing.T) {
	for _, named := range []struct {
		name  string
		model narration.Model
	}{
		{"opus", narration.Opus}, {" Sonnet ", narration.Sonnet}, {"haiku", narration.Haiku}, {"claude-opus-5-5", narration.Opus},
	} {
		known, err := narration.ModelOf(named.name)
		require.NoError(t, err, named.name)
		assert.Equal(t, named.model, known)
	}

	_, err := narration.ModelOf("gpt")
	assert.ErrorIs(t, err, narration.ErrUnknownModel)
}
