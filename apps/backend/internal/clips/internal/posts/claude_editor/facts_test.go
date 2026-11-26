package claude_editor_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/claude_editor"
)

func factsOf(t *testing.T, story posts.Story) map[string]any {
	t.Helper()

	text, err := claude_editor.FactsOf(offer.Findings, story)
	require.NoError(t, err)
	var facts map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &facts))
	return facts
}

func TestTheFactsOfAFrontAreItAloneAsTheEditorReadItWithTheNames(t *testing.T) {
	facts := factsOf(t, posts.Story{Front: 1})

	assert.ElementsMatch(t, []string{"now", "names", "front"}, keysOf(facts))
	assert.Equal(t, map[string]any{"fr": "France", "tr": "Turkey"}, facts["names"])
	assert.InDelta(t, 9844, facts["front"].(map[string]any)["tilesChanged"], 0)
	// How many played for each side: "ten Turks against one Bulgarian".
	assert.InDelta(t, 7, facts["front"].(map[string]any)["flags"].([]any)[0].(map[string]any)["players"], 0)
}

func TestTheFactsOfASagaHoldThePathsOfItsFlagsAlone(t *testing.T) {
	facts := factsOf(t, posts.Story{Saga: 1, Attacker: "cz"})

	require.Contains(t, facts, "saga")
	flags := facts["paths"].(map[string]any)["flags"].([]any)
	require.Len(t, flags, 1)
	assert.Equal(t, "cz", flags[0].(map[string]any)["flag"])
}

func TestAStoryTheFindingsDoNotHoldHasNoFacts(t *testing.T) {
	_, err := claude_editor.FactsOf(offer.Findings, posts.Story{Front: 9})
	require.ErrorIs(t, err, claude_editor.ErrNoSuchStory)

	_, err = claude_editor.FactsOf(offer.Findings, posts.Story{Seasonal: 1})
	require.ErrorIs(t, err, claude_editor.ErrNoSuchStory, "a day won is the generator's own words")
}

func keysOf(facts map[string]any) []string {
	keys := make([]string, 0, len(facts))
	for key := range facts {
		keys = append(keys, key)
	}
	return keys
}
