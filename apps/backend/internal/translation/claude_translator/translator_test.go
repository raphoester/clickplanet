package claude_translator_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/claude_translator"
)

// A stand-in for the Messages API: it keeps each request and answers each line it was sent with its key, reversed.
type fake struct {
	stop     string
	requests []map[string]any
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var request map[string]any
		assert.NoError(t, json.Unmarshal(body, &request))
		f.requests = append(f.requests, request)

		prompt := request["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
		var sent []struct {
			Key string `json:"key"`
		}
		_, list, _ := strings.Cut(prompt, "\n\n")
		assert.NoError(t, json.Unmarshal([]byte(list), &sent))
		answers := make([]map[string]string, 0, len(sent))
		for _, line := range sent {
			answers = append(answers, map[string]string{"key": line.Key, "text": "fr:" + line.Key})
		}
		text, err := json.Marshal(map[string]any{"lines": answers})
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": claude_translator.DefaultModel,
			"content":     []map[string]any{{"type": "text", "text": string(text)}},
			"stop_reason": f.stop,
			"usage":       map[string]any{"input_tokens": 100, "cache_read_input_tokens": 900, "cache_creation_input_tokens": 0, "output_tokens": 50},
		}))
	}))
	t.Cleanup(server.Close)
	return server
}

func translatorOf(t *testing.T, server *httptest.Server) *claude_translator.Translator {
	t.Helper()

	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: "sk-ant-test", BaseURL: server.URL}) //nolint:gosec // a test's key.
	require.NoError(t, err)
	glossary, err := translation.ReadGlossary()
	require.NoError(t, err)
	return claude_translator.New(client, claude_translator.DefaultModel, glossary)
}

func linesOf(count int) []translation.Line {
	lines := make([]translation.Line, 0, count)
	for i := range count {
		lines = append(lines, translation.Line{Key: translation.Key(fmt.Sprintf("k%d", i)), English: "Take a tile", Context: "a button"})
	}
	return lines
}

func TestEachLineIsAnsweredInBatchesThatShareTheGlossary(t *testing.T) {
	api := &fake{stop: "end_turn"}

	answers, spend, err := translatorOf(t, api.serve(t)).Translate(t.Context(), "fr", linesOf(70))

	require.NoError(t, err)
	assert.Len(t, answers, 70)
	assert.Equal(t, "fr:k69", answers["k69"])
	require.Len(t, api.requests, 2)
	assert.Equal(t, translation.Spend{InputTokens: 2000, OutputTokens: 100}, spend)
	assert.Equal(t, claude_translator.DefaultModel, api.requests[0]["model"])
	system := api.requests[0]["system"].([]any)
	assert.Equal(t, claude_translator.Instructions, system[0].(map[string]any)["text"])
	glossary := system[1].(map[string]any)
	assert.Contains(t, glossary["text"], "The glossary, in French (fr)")
	assert.Contains(t, glossary["text"], "- tile → case (")
	assert.NotNil(t, glossary["cache_control"])
	prompt := api.requests[0]["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"]
	assert.Contains(t, prompt, "Translate these 60 lines into French (fr).")
	assert.Contains(t, prompt, `"context": "a button"`)
}

func TestARefusalIsAnError(t *testing.T) {
	api := &fake{stop: "refusal"}

	_, _, err := translatorOf(t, api.serve(t)).Translate(t.Context(), "fr", linesOf(1))

	require.ErrorIs(t, err, claude_translator.ErrRefused)
}
