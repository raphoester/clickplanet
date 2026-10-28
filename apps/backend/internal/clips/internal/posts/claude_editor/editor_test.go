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
	at    = time.Date(2026, 10, 26, 20, 31, 0, 0, time.UTC)
	offer = posts.Offer{
		Count: 2, Now: at, Shown: map[string]int{"pw": 1},
		First: []int{1},
		Findings: posts.Findings{
			Since: at.Add(-12 * time.Hour), Until: at,
			Seasonal: []posts.Seasonal{{Title: "PALAU WINS DAY 18", Key: posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}}},
			Fronts: []posts.Front{{
				Since: at.Add(-5 * time.Hour), Until: at, TilesChanged: 9844, Captured: 9000,
				Countries: []posts.FrontCountry{{Code: "tr", Continent: "Middle East", Tiles: 2500, Changed: 6000, HoldersStart: map[string]int{"tr": 2000}, HoldersEnd: map[string]int{"fr": 1500}, Takers: map[string]int{"fr": 5000}}},
				Flags:     []posts.FrontFlag{{Code: "fr", Continent: "Europe", Took: 5000, Lost: 3000, TookFrom: map[string]int{"tr": 5000}, Solidity: 0.86}},
				Bombs:     3, Score: 3886.5,
			}},
			Tours:      []posts.Tour{{Flag: "fr", Since: at.Add(-12 * time.Hour), Until: at, Took: 27992, ByContinent: map[string]int{"Europe": 12000}, NextTook: 9000, FlippedByContinent: map[string]int{"South Asia": 6246}}},
			Places:     []posts.Place{{Kind: posts.ContinentPlace, Continent: "South Asia", Tiles: 9600, Holder: "fr", Held: 6170, Share: 0.643, Changed: 6000}},
			Names:      map[string]string{"fr": "France", "tr": "Turkey"},
			Continents: map[string]string{"Europe": "eu", "Middle East": ""},
			Anthems:    []string{"eu", "fr", "tr"},
		},
		Recent: []posts.Telling{{Title: "BRITTANY IS INVADING AUSTRALIA", Key: posts.StoryKey{Flag: "xb", Continent: "Oceania"}, Countries: []string{"au", "xb"}, At: at.Add(-time.Hour)}},
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
		`{"why":"The war is the day's story.","stories":[`+
			`{"front":1,"tour":0,"place":0,"kind":"battle","attacker":"fr","rival":"tr","takenFrom":["tr"],"countries":["tr"],"continent":"","team":"","headline":"FRANCE VS TURKEY","teasers":["THE WAR FOR TURKEY"],"line":"The battle for Turkey","call":"PICK A SIDE","callFlags":["fr","tr"],"anthem":"fr"},`+
			`{"front":0,"tour":1,"place":0,"kind":"","attacker":"","rival":"","takenFrom":[],"countries":[],"continent":"","team":"","headline":"FRANCE IS EVERYWHERE","teasers":["NOBODY CAN STOP FRANCE"],"line":"","call":"FIGHT BACK","callFlags":[],"anthem":""}]}`)}

	edition, spend, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.NoError(t, err)
	assert.Equal(t, posts.Edition{Why: "The war is the day's story.", Stories: []posts.Story{
		{
			Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY", Teasers: []string{"THE WAR FOR TURKEY"},
			Line: "The battle for Turkey", Call: "PICK A SIDE", CallFlags: []string{"fr", "tr"}, Anthem: "fr",
		},
		{
			Tour: 1, TakenFrom: []string{}, Countries: []string{}, Headline: "FRANCE IS EVERYWHERE", Teasers: []string{"NOBODY CAN STOP FRANCE"},
			Call: "FIGHT BACK", CallFlags: []string{},
		},
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
	story := format["schema"].(map[string]any)["properties"].(map[string]any)["stories"].(map[string]any)["items"].(map[string]any)
	assert.Equal(t, []any{"front", "tour", "place", "kind", "attacker", "rival", "takenFrom", "countries", "continent", "team", "headline", "teasers", "line", "call", "callFlags", "anthem"}, story["required"])

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

func TestThePromptHoldsTheFindingsNumberedTheStoriesToldFirstAndTheRecentPosts(t *testing.T) {
	prompt, err := claude_editor.PromptOf(offer)
	require.NoError(t, err)

	var written struct {
		Now        time.Time         `json:"now"`
		Slots      int               `json:"slots"`
		ToldFirst  []string          `json:"toldFirst"`
		Names      map[string]string `json:"names"`
		Continents []string          `json:"continents"`
		Anthems    []string          `json:"anthems"`
		Fronts     []struct {
			Front     int     `json:"front"`
			Hours     float64 `json:"hours"`
			Captured  int     `json:"captured"`
			Countries []struct {
				Code         string         `json:"code"`
				HoldersStart map[string]int `json:"holdersStart"`
				Takers       map[string]int `json:"takers"`
			} `json:"countries"`
			Flags []struct {
				Code     string         `json:"code"`
				TookFrom map[string]int `json:"tookFrom"`
				Solidity float64        `json:"solidity"`
			} `json:"flags"`
		} `json:"fronts"`
		Tours []struct {
			Tour               int            `json:"tour"`
			Flag               string         `json:"flag"`
			NextTook           int            `json:"nextTook"`
			FlippedByContinent map[string]int `json:"flippedByContinent"`
		} `json:"tours"`
		Places []struct {
			Place     int     `json:"place"`
			Kind      string  `json:"kind"`
			Continent string  `json:"continent"`
			Share     float64 `json:"share"`
		} `json:"places"`
		RecentPosts []struct {
			Headline  string   `json:"headline"`
			Countries []string `json:"countries"`
		} `json:"recentPosts"`
		ShownLately map[string]int `json:"shownLately"`
	}
	_, document, found := strings.Cut(prompt, "\n\n")
	require.True(t, found)
	require.NoError(t, json.Unmarshal([]byte(document), &written))

	assert.Equal(t, at, written.Now)
	assert.Equal(t, 1, written.Slots)
	assert.Equal(t, []string{"PALAU WINS DAY 18"}, written.ToldFirst)
	assert.Equal(t, map[string]string{"fr": "France", "tr": "Turkey"}, written.Names)
	assert.Equal(t, []string{"eu", "fr", "tr"}, written.Anthems)
	assert.Equal(t, []string{"Europe", "Middle East"}, written.Continents)
	require.Len(t, written.Fronts, 1)
	assert.Equal(t, 1, written.Fronts[0].Front)
	assert.InDelta(t, 5, written.Fronts[0].Hours, 0.01)
	assert.Equal(t, 9000, written.Fronts[0].Captured)
	assert.Equal(t, map[string]int{"tr": 2000}, written.Fronts[0].Countries[0].HoldersStart)
	assert.Equal(t, map[string]int{"fr": 5000}, written.Fronts[0].Countries[0].Takers)
	assert.Equal(t, map[string]int{"tr": 5000}, written.Fronts[0].Flags[0].TookFrom)
	assert.InDelta(t, 0.86, written.Fronts[0].Flags[0].Solidity, 1e-9)
	assert.Equal(t, 1, written.Tours[0].Tour)
	assert.Equal(t, 9000, written.Tours[0].NextTook)
	assert.Equal(t, map[string]int{"South Asia": 6246}, written.Tours[0].FlippedByContinent)
	assert.Equal(t, 1, written.Places[0].Place)
	assert.Equal(t, "South Asia", written.Places[0].Continent)
	assert.Equal(t, "BRITTANY IS INVADING AUSTRALIA", written.RecentPosts[0].Headline)
	assert.Equal(t, []string{"au", "xb"}, written.RecentPosts[0].Countries)
	assert.Equal(t, map[string]int{"pw": 1}, written.ShownLately)
}

func TestAPromptOver100KTokensIsBilledFiveTimesOver(t *testing.T) {
	short := claude_editor.SpendOf(claude_editor.DefaultModel, 100_000, 0, 0, 1000)
	long := claude_editor.SpendOf(claude_editor.DefaultModel, 100_001, 0, 0, 1000)

	assert.InDelta(t, 0.0105, short.USD, 1e-9)
	assert.InDelta(t, 5*(100_001*0.10+1000*0.50)/1e6, long.USD, 1e-9)
}
