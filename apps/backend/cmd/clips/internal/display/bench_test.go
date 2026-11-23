package display_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

func TestTheBenchFileHoldsEachScriptWithItsIDWhatItSaysAndHowItMeasures(t *testing.T) {
	round := clips.BenchRound{
		At:        time.Date(2026, 11, 20, 8, 24, 0, 0, time.UTC),
		Stories:   []clips.Story{{Saga: 2, Attacker: "pw", Rival: "ru", Headline: "TINY PALAU RULES RUSSIA"}},
		Editorial: clips.Editorial{Asked: true, Why: "Palau", Spend: clips.Spend{USD: 0.002}},
		Entries: []clips.BenchEntry{
			{
				Story: 1, Tone: "deadpan", Model: clips.Opus, Spend: clips.Spend{USD: 0.05},
				Script: clips.Script{Title: "DAY 9: PALAU STILL OWNS RUSSIA", Lines: []string{"Day nine.", "Palau still owns Russia!"}},
			},
			{Story: 1, Tone: "storyteller", Model: clips.Haiku, Failed: "the model refused (general_harms)"},
		},
	}
	file := filepath.Join(t.TempDir(), "bench.json")
	var out bytes.Buffer

	require.NoError(t, display.Bench(file, &out, round))

	raw, err := os.ReadFile(file) //nolint:gosec // the file the test wrote.
	require.NoError(t, err)
	var written map[string]any
	require.NoError(t, json.Unmarshal(raw, &written))
	assert.InDelta(t, 0.052, written["usd"], 1e-9)
	assert.Equal(t, []any{map[string]any{"story": float64(1), "of": "saga 2", "attacker": "pw", "rival": "ru", "headline": "TINY PALAU RULES RUSSIA"}}, written["stories"])
	entries := written["entries"].([]any)
	require.Len(t, entries, 2)
	assert.Equal(t, map[string]any{
		"id": "1-deadpan-opus", "story": float64(1), "tone": "deadpan", "model": "claude-opus-5-5",
		"title": "DAY 9: PALAU STILL OWNS RUSSIA", "lines": []any{"Day nine.", "Palau still owns Russia!"},
		"said": "Day nine. Palau still owns Russia!", "words": float64(6), "seconds": 1.8, "tells": []any{}, "usd": 0.05,
	}, entries[0])
	assert.Equal(t, "1-storyteller-haiku", entries[1].(map[string]any)["id"])
	assert.Equal(t, "the model refused (general_harms)", entries[1].(map[string]any)["failed"])

	assert.Equal(t, "1-deadpan-opus: 6 words, 1.8 s, 0 tells []\n1-storyteller-haiku: failed: the model refused (general_harms)\n"+
		"2 scripts of 1 stories, about $0.052\n", out.String())
}
