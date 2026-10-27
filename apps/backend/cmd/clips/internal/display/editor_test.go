package display_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

func summaryOf(t *testing.T, editorial clips.Editorial) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "summary.md")
	require.NoError(t, display.Editorial(path, editorial))
	written, err := os.ReadFile(path) //nolint:gosec // the file the test wrote.
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(t, err)
	return string(written)
}

var spend = clips.SpendOf(3000, 400)

func TestTheSummarySaysTheModelChoseTheRunAndWhatItCost(t *testing.T) {
	spend := spend
	spend.CacheWriteTokens = 1800

	summary := summaryOf(t, clips.Editorial{Asked: true, Why: "Brazil is fresher.", Spend: spend})

	assert.Equal(t, "**Editor: the model.** Brazil is fresher. `claude-haiku-5-5`: 4800 tokens in (1800 written to the cache, "+
		"0 read from it), 400 out, about $0.0005.\n\n", summary)
}

func TestTheSummarySaysWhyTheEditorGaveNoAnswer(t *testing.T) {
	summary := summaryOf(t, clips.Editorial{Asked: true, Failed: "the model refused (general_harms)", Spend: spend})

	assert.Contains(t, summary, "**Editor: no answer, the run is its day won alone.** the model refused (general_harms). `claude-haiku-5-5`")
}

func TestTheSummarySaysWhenNothingWasAsked(t *testing.T) {
	assert.Equal(t, "**Editor: the model**, with nothing to decide: no call.\n\n", summaryOf(t, clips.Editorial{}))
}

func TestNoSummaryFileIsNoSummary(t *testing.T) {
	require.NoError(t, display.Editorial("", clips.Editorial{Asked: true, Spend: spend}))
}

func TestATrialOfThePromptAlonePrintsItAndNoChoice(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Trial(&out, clips.Trial{Instructions: "be an editor", Prompt: "the run", Tokens: 2400}, true))

	assert.Contains(t, out.String(), "be an editor\n=== the run ===\nthe run\n")
	assert.Contains(t, out.String(), "The prompt is 2400 tokens, about $0.00024 in at most.")
	assert.NotContains(t, out.String(), "**Editor")
}

func TestATrialPrintsEachStoryChosenAsTheGeneratorTakesItAndEachSkippedWithWhy(t *testing.T) {
	var out bytes.Buffer
	trial := clips.Trial{
		Instructions: "be an editor", Prompt: "the run",
		Offer: clips.Offer{Findings: clips.Findings{Seasonal: []clips.Seasonal{{Title: "PALAU WINS DAY 18"}}}},
		Choice: clips.Choice{
			Chosen: []clips.Chosen{
				{Pick: 1, Story: clips.Story{Seasonal: 1}},
				{Pick: 2, Story: clips.Story{Front: 3, Kind: "battle", Attacker: "fr", Rival: "tr", Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY"}},
			},
			Skipped:   []clips.Skipped{{Story: clips.Story{Front: 9}, Why: "no front 9 of 3"}},
			Editorial: clips.Editorial{Asked: true, Why: "the war", Spend: spend},
		},
	}

	require.NoError(t, display.Trial(&out, trial, false))

	assert.Contains(t, out.String(), "**Editor: the model.** the war.")
	assert.Contains(t, out.String(), "skipped front 9: no front 9 of 3\n")
	assert.Contains(t, out.String(), "#1 PALAU WINS DAY 18\n  --story '{\"seasonal\":1}'\n")
	assert.Regexp(t, `#2 FRANCE VS TURKEY\n  --story '\{"front": ?3, ?"kind": ?"battle"`, out.String())
}
