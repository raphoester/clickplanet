package posts_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAScriptsTitleIsInCapitalsAndEachLineOnOneLine(t *testing.T) {
	script, err := posts.ScriptOf(" Day 25: Palau still rules Russia ", []string{"Day twenty five.", "", "Palau  still\nrules Russia."})

	require.NoError(t, err)
	assert.Equal(t, posts.Script{Title: "DAY 25: PALAU STILL RULES RUSSIA", Lines: []string{"Day twenty five.", "Palau still rules Russia."}}, script)
	assert.True(t, script.Voiced())
}

func TestAScriptWithNoTitleOrOneTooLongForYouTubeCannotBeVoiced(t *testing.T) {
	for _, title := range []string{"", "  ", strings.Repeat("A", 87)} {
		_, err := posts.ScriptOf(title, []string{"Day twenty five."})

		require.ErrorIs(t, err, posts.ErrInvalidScript, title)
	}
}

func TestAScriptThatSaysNothingCannotBeVoiced(t *testing.T) {
	_, err := posts.ScriptOf("DAY 25", []string{" ", ""})

	require.ErrorIs(t, err, posts.ErrInvalidScript)
	assert.False(t, posts.Script{}.Voiced())
}

func TestAScriptLongerThanAClipOf30SecondsSaysCannotBeVoiced(t *testing.T) {
	_, err := posts.ScriptOf("DAY 25", []string{strings.Repeat("word ", 80)})
	require.NoError(t, err, "80 words fit")

	_, err = posts.ScriptOf("DAY 25", []string{strings.Repeat("word ", 40), strings.Repeat("word ", 41)})
	require.ErrorIs(t, err, posts.ErrInvalidScript)
}
