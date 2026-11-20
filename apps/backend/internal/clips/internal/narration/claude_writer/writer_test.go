package claude_writer_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration/claude_writer"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

const key = "sk-ant-test-key"

var (
	deadpan = narration.Tone{Name: "deadpan", Brief: "# Deadpan\n\nSay it flat."}
	request = narration.Request{
		Story: posts.Story{
			Saga: 2, Since: time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC), Attacker: "pw", Rival: "ru",
			Headline: "TINY PALAU RULES RUSSIA",
		},
		Facts: `{"saga":{"saga":2,"code":"ru"},"names":{"pw":"Palau","ru":"Russia"}}`, Tone: deadpan, Day: 9,
	}
)

// A stand-in for the Messages API: it keeps each request and answers what it is given.
type fake struct {
	answer   map[string]any
	requests []map[string]any
	headers  []http.Header
}

func (f *fake) client(t *testing.T) *claude_writer.Writer {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var sent map[string]any
		assert.NoError(t, json.Unmarshal(body, &sent))
		f.requests = append(f.requests, sent)
		f.headers = append(f.headers, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(f.answer))
	}))
	t.Cleanup(server.Close)

	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: server.URL})
	require.NoError(t, err)
	return claude_writer.New(client)
}

func message(stop, text string) map[string]any {
	return map[string]any{
		"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
		"content":     []map[string]any{{"type": "thinking", "thinking": "", "signature": "sig"}, {"type": "text", "text": text}},
		"stop_reason": stop,
		"usage":       map[string]any{"input_tokens": 3000, "cache_creation_input_tokens": 1000, "cache_read_input_tokens": 0, "output_tokens": 2000},
	}
}

const answered = `{"why":"Say the absurd flat.","title":"DAY 9: PALAU STILL OWNS RUSSIA",` +
	`"lines":["Day nine.","Palau still owns  Russia.","Russia has not commented."]}`

func TestTheScriptIsTheModelsLinesWithItsTitleAndWhy(t *testing.T) {
	api := &fake{answer: message("end_turn", answered)}

	script, spend, err := api.client(t).Write(t.Context(), narration.Opus, request)

	require.NoError(t, err)
	assert.Equal(t, narration.Script{
		Title: "DAY 9: PALAU STILL OWNS RUSSIA", Why: "Say the absurd flat.",
		Lines: []string{"Day nine.", "Palau still owns Russia.", "Russia has not commented."},
	}, script)
	// 3,000 in at $4, 1,000 written to the cache at $5, 2,000 out at $20, a million each.
	assert.InDelta(t, 0.012+0.005+0.04, spend.USD, 1e-9)
	assert.Equal(t, "claude-opus-5-5", spend.Model)
}

func TestTheRequestCachesTheInstructionsThenGivesTheToneTheEffortAndTheAnswersShape(t *testing.T) {
	api := &fake{answer: message("end_turn", answered)}

	_, _, err := api.client(t).Write(t.Context(), narration.Opus, request)

	require.NoError(t, err)
	require.Len(t, api.requests, 1)
	sent := api.requests[0]
	assert.Equal(t, "claude-opus-5-5", sent["model"])
	assert.Equal(t, []any{
		map[string]any{"type": "text", "text": claude_writer.Instructions, "cache_control": map[string]any{"type": "ephemeral"}},
		map[string]any{"type": "text", "text": deadpan.Brief},
	}, sent["system"])
	config := sent["output_config"].(map[string]any)
	assert.Equal(t, "high", config["effort"])
	assert.Equal(t, []any{"why", "title", "lines"}, config["format"].(map[string]any)["schema"].(map[string]any)["required"])
	assert.Equal(t, "default", sent["fallbacks"], "a refusal on Opus is served again by Anthropic's fallback")
	assert.Contains(t, api.headers[0].Get("Anthropic-Beta"), "server-side-fallback-2026-07-01")
	assert.NotContains(t, sent, "thinking", "adaptive by default on every writer")

	prompt, err := claude_writer.PromptOf(request)
	require.NoError(t, err)
	assert.Equal(t, prompt, sent["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"])
}

func TestHaikuHasNoFallbackToAskFor(t *testing.T) {
	api := &fake{answer: message("end_turn", answered)}

	_, spend, err := api.client(t).Write(t.Context(), narration.Haiku, request)

	require.NoError(t, err)
	assert.NotContains(t, api.requests[0], "fallbacks")
	assert.Empty(t, api.headers[0].Get("Anthropic-Beta"))
	assert.InDelta(t, (3000*0.10+1000*0.125+2000*0.50)/1_000_000, spend.USD, 1e-12)
}

func TestThePromptIsTheStoryItsFactsTheDayAndTheEpisodesBefore(t *testing.T) {
	prompt, err := claude_writer.PromptOf(request)
	require.NoError(t, err)

	var written map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(prompt, "The story to tell, as JSON:\n\n")), &written))
	assert.Equal(t, map[string]any{
		"story": map[string]any{"attacker": "pw", "rival": "ru", "since": "2026-11-18T09:00:00Z", "headline": "TINY PALAU RULES RUSSIA"},
		"facts": map[string]any{"saga": map[string]any{"saga": float64(2), "code": "ru"}, "names": map[string]any{"pw": "Palau", "ru": "Russia"}},
		"day":   float64(9), "episodes": []any{},
	}, written)

	_, err = claude_writer.PromptOf(narration.Request{Facts: "not json"})
	assert.ErrorIs(t, err, claude_writer.ErrNoAnswer)
}

func TestARefusalIsNoScriptButItsCostIsKept(t *testing.T) {
	refused := message("refusal", "")
	refused["stop_details"] = map[string]any{"type": "refusal", "category": "general_harms", "explanation": ""}
	api := &fake{answer: refused}

	_, spend, err := api.client(t).Write(t.Context(), narration.Sonnet, request)

	require.ErrorIs(t, err, claude_writer.ErrRefused)
	assert.Contains(t, err.Error(), "general_harms")
	assert.Positive(t, spend.USD)
}

func TestAnAnswerWithNoLineIsNoScript(t *testing.T) {
	api := &fake{answer: message("end_turn", `{"why":"","title":"T","lines":["  "]}`)}

	_, _, err := api.client(t).Write(t.Context(), narration.Opus, request)

	assert.ErrorIs(t, err, claude_writer.ErrNoAnswer)
}
