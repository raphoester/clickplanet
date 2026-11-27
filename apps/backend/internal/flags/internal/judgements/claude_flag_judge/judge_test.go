package claude_flag_judge_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements/claude_flag_judge"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

const key = "sk-ant-test-key"

type fake struct {
	answer   map[string]any
	status   int
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
		w.Header().Set("Content-Type", "application/json")
		if f.status != 0 {
			w.WriteHeader(f.status)
		}
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
		"usage":       map[string]any{"input_tokens": 2100, "output_tokens": 80},
	}
}

func judgeOf(t *testing.T, server *httptest.Server) *claude_flag_judge.Judge {
	t.Helper()
	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: server.URL})
	require.NoError(t, err)
	return claude_flag_judge.New(client, "")
}

var bavaria = judgements.Case{
	Name: "Bavaria", Pin: catalog.Pin{Latitude: 48.1374, Longitude: 11.5755}, Picture: []byte("png"),
	Taken: []catalog.Name{"Brittany", "Tyrol"},
}

func TestTheRequestSendsTheRuleThePictureAndTheCase(t *testing.T) {
	api := &fake{answer: message("end_turn", `{"verdict":"approve","reason":"","note":"","why":"Bavaria's lozenges"}`)}

	ruling, err := judgeOf(t, api.serve(t)).Judge(t.Context(), bavaria)
	require.NoError(t, err)
	assert.Equal(t, judgements.Ruling{Verdict: judgements.Approve, Why: "Bavaria's lozenges"}, ruling)

	require.Len(t, api.requests, 1)
	sent := api.requests[0]
	assert.Equal(t, "claude-haiku-5-5", sent["model"])
	assert.Equal(t, claude_flag_judge.Instructions, sent["system"].([]any)[0].(map[string]any)["text"])
	content := sent["messages"].([]any)[0].(map[string]any)["content"].([]any)
	image := content[0].(map[string]any)["source"].(map[string]any)
	assert.Equal(t, "image/png", image["media_type"])
	assert.Equal(t, "cG5n", image["data"], "the picture, base64")
	assert.JSONEq(t, `{"name":"Bavaria","capital":{"latitude":48.1374,"longitude":11.5755},
		"flags_in_the_game":["Brittany","Tyrol"]}`,
		content[1].(map[string]any)["text"].(string)[len("The flag sent, its picture above:\n"):])
}

func TestARejectionCarriesItsReasonAndTheNoteThePlayerReads(t *testing.T) {
	api := &fake{answer: message("end_turn",
		`{"verdict":"reject","reason":"not_a_place","note":"A football club is not a state.","why":"a club's crest"}`)}

	ruling, err := judgeOf(t, api.serve(t)).Judge(t.Context(), bavaria)
	require.NoError(t, err)

	assert.Equal(t, judgements.Reject, ruling.Verdict)
	assert.Equal(t, submissions.Rejection{Reason: submissions.NotAPlace, Note: "A football club is not a state."},
		ruling.Rejection)
}

func TestWhatTheGameCannotActOnIsLeftToTheOperator(t *testing.T) {
	for name, answer := range map[string]map[string]any{
		"a refusal":               message("refusal", ""),
		"cut short":               message("max_tokens", `{"verdict":"appr`),
		"no JSON":                 message("end_turn", "Approve it."),
		"other with no note":      message("end_turn", `{"verdict":"reject","reason":"other","note":"","why":"odd"}`),
		"a reason the game lacks": message("end_turn", `{"verdict":"reject","reason":"ugly","note":"","why":"ugly"}`),
	} {
		api := &fake{answer: answer}

		ruling, err := judgeOf(t, api.serve(t)).Judge(t.Context(), bavaria)

		require.NoError(t, err, name)
		assert.Equal(t, judgements.Unsure, ruling.Verdict, name)
	}
}

func TestAnAPIThatFailsIsAnErrorToTryAgain(t *testing.T) {
	api := &fake{status: http.StatusServiceUnavailable, answer: map[string]any{
		"type": "error", "error": map[string]any{"type": "overloaded_error", "message": "Overloaded"},
	}}

	_, err := judgeOf(t, api.serve(t)).Judge(t.Context(), bavaria)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "sk-ant", "no key in a log")
}
