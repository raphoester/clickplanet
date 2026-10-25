package claude_editor_test

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

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/claude_editor"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

const key = "sk-ant-test-key"

var (
	at    = time.Date(2026, 10, 24, 12, 17, 0, 0, time.UTC)
	offer = posts.Offer{
		Count: 2, Now: at, Shown: map[string]int{"pw": 1},
		Stories: posts.Shortlist{
			{
				Pick: 2, Title: "FRANCE VS GERMANY", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}, Countries: []string{"fr", "de"},
				Teasers: []string{"THE BATTLE FOR FRANCE"}, Facts: posts.StoryFacts(`{"kind":"battle","storyTilesChanged":1200}`),
			},
			{Pick: 4, Title: "BRAZIL IS INVADING ARGENTINA", Key: posts.StoryKey{Flag: "br", Continent: "South America"}},
		},
		Recent: []posts.Telling{{Title: "PALAU IS EVERYWHERE", Key: posts.StoryKey{Flag: "pw", Continent: "Asia"}, At: at.Add(-time.Hour)}},
	}
)

// A stand-in for the Messages API: it keeps each request and answers what it is given.
type fake struct {
	status   int
	answer   map[string]any
	requests []map[string]any
	headers  []http.Header
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var request map[string]any
		assert.NoError(t, json.Unmarshal(body, &request))
		f.requests = append(f.requests, request)
		f.headers = append(f.headers, r.Header.Clone())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(f.status)
		assert.NoError(t, json.NewEncoder(w).Encode(f.answer))
	}))
	t.Cleanup(server.Close)
	return server
}

func message(stop string, text string) map[string]any {
	return map[string]any{
		"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-haiku-5-5",
		"content": []map[string]any{
			{"type": "thinking", "thinking": "", "signature": "sig"},
			{"type": "text", "text": text},
		},
		"stop_reason": stop,
		"usage": map[string]any{
			"input_tokens": 1500, "cache_creation_input_tokens": 2000, "cache_read_input_tokens": 0, "output_tokens": 400,
		},
	}
}

func editorOf(t *testing.T, server *httptest.Server) *claude_editor.Editor {
	t.Helper()

	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: server.URL})
	require.NoError(t, err)
	return claude_editor.New(client, claude_editor.DefaultModel)
}

func TestTheModelsAnswerIsTheEditionAndItsCost(t *testing.T) {
	api := &fake{status: http.StatusOK, answer: message("end_turn",
		`{"why":"Brazil is fresher.","stories":[{"pick":4,"headline":"BRAZIL STORMS ARGENTINA"},{"pick":2,"headline":"FRANCE VS GERMANY"}]}`)}

	edition, spend, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.NoError(t, err)
	assert.Equal(t, posts.Edition{Why: "Brazil is fresher.", Edits: []posts.Edit{
		{Pick: 4, Headline: "BRAZIL STORMS ARGENTINA"}, {Pick: 2, Headline: "FRANCE VS GERMANY"},
	}}, edition)
	assert.Equal(t, int64(1500), spend.InputTokens)
	assert.Equal(t, int64(2000), spend.CacheWriteTokens)
	assert.Equal(t, int64(400), spend.OutputTokens)
	// 1,500 in at $0.10, 2,000 written to the cache at $0.125, 400 out at $0.50, a million each.
	assert.InDelta(t, 0.00015+0.00025+0.0002, spend.USD, 1e-9)
}

func TestTheRequestCachesTheInstructionsAndAsksForTheAnswersShape(t *testing.T) {
	api := &fake{status: http.StatusOK, answer: message("end_turn", `{"why":"","stories":[]}`)}

	_, _, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.NoError(t, err)
	require.Len(t, api.requests, 1)
	request := api.requests[0]
	assert.Equal(t, "claude-haiku-5-5", request["model"])
	assert.Equal(t, []any{map[string]any{
		"type": "text", "text": claude_editor.Instructions, "cache_control": map[string]any{"type": "ephemeral"},
	}}, request["system"])
	assert.Equal(t, "medium", request["output_config"].(map[string]any)["effort"])
	format := request["output_config"].(map[string]any)["format"].(map[string]any)
	assert.Equal(t, "json_schema", format["type"])
	assert.Equal(t, []any{"why", "stories"}, format["schema"].(map[string]any)["required"])

	prompt, err := claude_editor.PromptOf(offer)
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": prompt}}}},
		request["messages"])
	assert.Equal(t, key, api.headers[0].Get("X-Api-Key"))
}

func TestARefusalIsNoEditionButItsCostIsKept(t *testing.T) {
	refused := message("refusal", "")
	refused["stop_details"] = map[string]any{"type": "refusal", "category": "general_harms", "explanation": "no"}
	api := &fake{status: http.StatusOK, answer: refused}

	_, spend, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.ErrorIs(t, err, claude_editor.ErrRefused)
	assert.Contains(t, err.Error(), "general_harms")
	assert.Equal(t, int64(400), spend.OutputTokens)
}

func TestAnAnswerCutShortOrNotJSONIsNoEdition(t *testing.T) {
	for name, answer := range map[string]map[string]any{
		"cut short": message("max_tokens", `{"why":"Bra`),
		"not JSON":  message("end_turn", "Brazil, then France."),
	} {
		t.Run(name, func(t *testing.T) {
			api := &fake{status: http.StatusOK, answer: answer}

			_, _, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

			require.ErrorIs(t, err, claude_editor.ErrNoAnswer)
		})
	}
}

// The logs of a public repo are public: an error says what went wrong, never the request it carried.
func TestAnAPIErrorSaysItsStatusAndNeverTheKey(t *testing.T) {
	api := &fake{status: http.StatusBadRequest, answer: map[string]any{
		"type": "error", "error": map[string]any{"type": "invalid_request_error", "message": "bad"},
	}}

	_, spend, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
	assert.NotContains(t, err.Error(), key)
	assert.Equal(t, posts.Spend{Model: claude_editor.DefaultModel}, spend, "sent, at no cost known")
}

func TestWithNoKeyNothingIsSentAndTheRunFallsBack(t *testing.T) {
	_, spend, err := claude_editor.Unkeyed{}.Edit(t.Context(), offer)
	_, countErr := claude_editor.Unkeyed{}.Count(t.Context(), offer)

	require.ErrorIs(t, err, cpanthropic.ErrNoKey)
	require.ErrorIs(t, countErr, cpanthropic.ErrNoKey)
	assert.Equal(t, posts.Spend{}, spend, "nothing sent, nothing asked")
}

func TestThePromptHoldsTheStoriesTheirFactsAndTheRecentPosts(t *testing.T) {
	prompt, err := claude_editor.PromptOf(offer)
	require.NoError(t, err)

	var written struct {
		Now     time.Time `json:"now"`
		Slots   int       `json:"slots"`
		Stories []struct {
			Pick     int             `json:"pick"`
			Headline string          `json:"headline"`
			Teasers  []string        `json:"teasers"`
			Facts    json.RawMessage `json:"facts"`
		} `json:"stories"`
		RecentPosts []struct {
			Headline string `json:"headline"`
		} `json:"recentPosts"`
		ShownLately map[string]int `json:"shownLately"`
	}
	_, document, found := strings.Cut(prompt, "\n\n")
	require.True(t, found)
	require.NoError(t, json.Unmarshal([]byte(document), &written))

	assert.Equal(t, at, written.Now)
	assert.Equal(t, 2, written.Slots)
	require.Len(t, written.Stories, 2)
	assert.Equal(t, 2, written.Stories[0].Pick)
	assert.Equal(t, []string{"THE BATTLE FOR FRANCE"}, written.Stories[0].Teasers)
	assert.JSONEq(t, `{"kind":"battle","storyTilesChanged":1200}`, string(written.Stories[0].Facts))
	assert.Empty(t, written.Stories[1].Facts)
	assert.Equal(t, "PALAU IS EVERYWHERE", written.RecentPosts[0].Headline)
	assert.Equal(t, map[string]int{"pw": 1}, written.ShownLately)
}

func TestAPromptOver100KTokensIsBilledFiveTimesOver(t *testing.T) {
	short := claude_editor.SpendOf(claude_editor.DefaultModel, 100_000, 0, 0, 1000)
	long := claude_editor.SpendOf(claude_editor.DefaultModel, 100_001, 0, 0, 1000)

	assert.InDelta(t, 0.0105, short.USD, 1e-9)
	assert.InDelta(t, 5*(100_001*0.10+1000*0.50)/1e6, long.USD, 1e-9)
}
