package narration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

type writer struct {
	script   narration.Script
	spend    posts.Spend
	err      error
	models   []narration.Model
	requests []narration.Request
}

func (w *writer) Write(_ context.Context, model narration.Model, request narration.Request) (narration.Script, posts.Spend, error) {
	w.models = append(w.models, model)
	w.requests = append(w.requests, request)
	return w.script, w.spend, w.err
}

var (
	storyteller = narration.Tone{Name: "storyteller", Brief: "Tell it to a friend."}
	palau       = posts.Story{Saga: 5, Attacker: "pw", Headline: "PALAU RULES RUSSIA", Experiment: "war-report"}
	at          = time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
)

func factsOf(_ posts.Findings, story posts.Story) (string, error) {
	return fmt.Sprintf(`{"saga":%d}`, story.Saga), nil
}

func TestTheWarReportIsWrittenInItsToneByItsModelOnTheFindingsDayAfterTheEpisodesBefore(t *testing.T) {
	written := &writer{
		script: narration.Script{Script: posts.Script{Title: "Day 25: Palau still rules Russia", Lines: []string{"Day twenty five.", "Palau still rules Russia."}}, Why: "the reign"},
		spend:  posts.Spend{Model: "claude-sonnet-5-5", USD: 0.01},
	}
	told := []posts.Telling{
		{At: at.Add(-2 * time.Hour), Said: []string{"Day twenty five.", "Egypt hits back."}},
		{At: at.Add(-30 * time.Hour), Said: []string{"Day twenty four.", "Palau takes Russia."}},
		{At: at.Add(-time.Hour), Title: "FRANCE VS TURKEY"},
	}

	script, spend, err := narration.NewNarrator(written, factsOf, storyteller, narration.Sonnet).
		Narrate(t.Context(), posts.Findings{Day: 25}, palau, told)

	require.NoError(t, err)
	assert.Equal(t, narration.Script{
		Script: posts.Script{Title: "DAY 25: PALAU STILL RULES RUSSIA", Lines: []string{"Day twenty five.", "Palau still rules Russia."}},
		Why:    "the reign",
	}, script)
	assert.Equal(t, written.spend, spend)
	assert.Equal(t, []narration.Model{narration.Sonnet}, written.models)
	assert.Equal(t, []narration.Request{{
		Story: palau, Facts: `{"saga":5}`, Tone: storyteller, Day: 25,
		Episodes: []string{"Day twenty four. Palau takes Russia.", "Day twenty five. Egypt hits back."},
	}}, written.requests)
}

func TestAScriptThatCannotBeVoicedIsAnErrorWithItsCostKept(t *testing.T) {
	written := &writer{
		script: narration.Script{Script: posts.Script{Title: "DAY 25", Lines: []string{strings.Repeat("word ", 90)}}},
		spend:  posts.Spend{Model: "claude-sonnet-5-5", USD: 0.02},
	}

	_, spend, err := narration.NewNarrator(written, factsOf, storyteller, narration.Sonnet).Narrate(t.Context(), posts.Findings{}, palau, nil)

	require.ErrorIs(t, err, posts.ErrInvalidScript)
	assert.Equal(t, written.spend, spend)
}

func TestAWriterThatFailsIsAnErrorWithItsCostKept(t *testing.T) {
	refused := errors.New("the model refused (general_harms)")
	written := &writer{err: refused, spend: posts.Spend{Model: "claude-sonnet-5-5", USD: 0.002}}

	_, spend, err := narration.NewNarrator(written, factsOf, storyteller, narration.Sonnet).Narrate(t.Context(), posts.Findings{}, palau, nil)

	require.ErrorIs(t, err, refused)
	assert.Equal(t, written.spend, spend)
}

func TestFactsThatCannotBeReadAskNoWriter(t *testing.T) {
	written := &writer{}
	unread := func(posts.Findings, posts.Story) (string, error) { return "", errors.New("a seasonal story has none") }

	_, _, err := narration.NewNarrator(written, unread, storyteller, narration.Sonnet).Narrate(t.Context(), posts.Findings{}, palau, nil)

	require.ErrorContains(t, err, "a seasonal story has none")
	assert.Empty(t, written.requests)
}

func TestTheEpisodesAreTheLastTenWarReportsOldestFirst(t *testing.T) {
	told := make([]posts.Telling, 0, 13)
	for i := range 12 {
		told = append(told, posts.Telling{At: at.Add(-time.Duration(i) * time.Hour), Said: []string{fmt.Sprintf("Episode %d.", 12-i), "Said."}})
	}
	told = append(told, posts.Telling{At: at, Title: "A CLIP WITH NO VOICE"})

	episodes := narration.EpisodesOf(told)

	require.Len(t, episodes, 10)
	assert.Equal(t, "Episode 3. Said.", episodes[0])
	assert.Equal(t, "Episode 12. Said.", episodes[9])
	assert.Empty(t, narration.EpisodesOf(nil))
}
