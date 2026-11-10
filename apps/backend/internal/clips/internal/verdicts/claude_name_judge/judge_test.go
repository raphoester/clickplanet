package claude_name_judge_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/claude_name_judge"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

const key = "sk-ant-test-key"

type fake struct {
	answers  []map[string]any
	requests []map[string]any
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var request map[string]any
		assert.NoError(t, json.Unmarshal(body, &request))
		answer := f.answers[min(len(f.requests), len(f.answers)-1)]
		f.requests = append(f.requests, request)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(answer))
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
		"usage":       map[string]any{"input_tokens": 900, "output_tokens": 200},
	}
}

func judgeOf(t *testing.T, server *httptest.Server) *claude_name_judge.Judge {
	t.Helper()
	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: key, BaseURL: server.URL})
	require.NoError(t, err)
	return claude_name_judge.New(client, claude_name_judge.DefaultModel)
}

func TestEachNumberAnsweredIsTheJudgementOfItsName(t *testing.T) {
	api := &fake{answers: []map[string]any{message("end_turn", `{"names":[
		{"number":2,"shown":false,"reason":"the IRA, an armed group"},
		{"number":1,"shown":true,"reason":""},
		{"number":2,"shown":true,"reason":""},
		{"number":7,"shown":true,"reason":""}
	]}`)}}

	judged, err := judgeOf(t, api.serve(t)).Judge(t.Context(), []string{"Ada", "IRA4Ever", "Bob"})

	require.NoError(t, err)
	assert.Equal(t, []verdicts.Judgement{
		{Name: "IRA4Ever", Shown: false, Reason: "the IRA, an armed group"},
		{Name: "Ada", Shown: true},
	}, judged, "a number answered twice counts once, one it was not given not at all, and Bob is left out")
}

func TestTheRequestSendsTheInstructionsAndTheNamesAsJSON(t *testing.T) {
	api := &fake{answers: []map[string]any{message("end_turn", `{"names":[]}`)}}

	_, err := judgeOf(t, api.serve(t)).Judge(t.Context(), []string{`Ada "the" Great`})

	require.NoError(t, err)
	require.Len(t, api.requests, 1)
	request := api.requests[0]
	assert.Equal(t, "claude-haiku-5-5", request["model"])
	assert.Equal(t, []any{map[string]any{"type": "text", "text": claude_name_judge.Instructions}}, request["system"])
	prompt, err := claude_name_judge.PromptOf([]string{`Ada "the" Great`})
	require.NoError(t, err)
	assert.Equal(t, "The names to judge:\n"+`[{"number":1,"name":"Ada \"the\" Great"}]`, prompt)
	config := request["output_config"].(map[string]any) //nolint:forcetypeassert // the request's shape is the test.
	assert.Equal(t, "medium", config["effort"])
	format := config["format"].(map[string]any) //nolint:forcetypeassert // the request's shape is the test.
	assert.Equal(t, "json_schema", format["type"])
}

func TestManyNamesAreJudgedAHundredACall(t *testing.T) {
	names := make([]string, 150)
	for i := range names {
		names[i] = fmt.Sprintf("Player%d", i)
	}
	api := &fake{answers: []map[string]any{message("end_turn", `{"names":[{"number":1,"shown":true,"reason":""}]}`)}}

	judged, err := judgeOf(t, api.serve(t)).Judge(t.Context(), names)

	require.NoError(t, err)
	assert.Len(t, api.requests, 2)
	assert.Equal(t, []verdicts.Judgement{{Name: "Player0", Shown: true}, {Name: "Player100", Shown: true}}, judged)
}

func TestARefusalJudgesNothing(t *testing.T) {
	refusal := message("refusal", "")
	refusal["stop_details"] = map[string]any{"type": "refusal", "category": "general_harms"}
	api := &fake{answers: []map[string]any{refusal}}

	_, err := judgeOf(t, api.serve(t)).Judge(t.Context(), []string{"Ada"})

	require.ErrorIs(t, err, claude_name_judge.ErrRefused)
	assert.Contains(t, err.Error(), "general_harms")
}

func TestAnAnswerCutShortJudgesNothing(t *testing.T) {
	api := &fake{answers: []map[string]any{message("max_tokens", `{"names":[`)}}

	_, err := judgeOf(t, api.serve(t)).Judge(t.Context(), []string{"Ada"})

	require.ErrorIs(t, err, claude_name_judge.ErrNoAnswer)
}

func TestWithNoKeyNoNameIsJudged(t *testing.T) {
	_, err := claude_name_judge.Unkeyed{}.Judge(t.Context(), []string{"Ada"})

	require.ErrorIs(t, err, verdicts.ErrNoJudge)
}
