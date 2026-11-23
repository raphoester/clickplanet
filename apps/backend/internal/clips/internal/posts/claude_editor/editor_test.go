package claude_editor_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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
				Flags: []posts.FrontFlag{{
					Code: "fr", Continent: "Europe", Took: 5000, Lost: 3000, TookFrom: map[string]int{"tr": 5000}, Solidity: 0.86,
					FlippedByContinent: map[string]int{"Middle East": 2400},
					Fighters:           []posts.Fighter{{Number: 1, Name: "BraveFox42", Took: 812}},
				}},
				Bombs: 3, Score: 3886.5,
			}},
			Tours:  []posts.Tour{{Flag: "fr", Since: at.Add(-12 * time.Hour), Until: at, Took: 27992, ByContinent: map[string]int{"Europe": 12000}, NextTook: 9000, FlippedByContinent: map[string]int{"South Asia": 6246}}},
			Places: []posts.Place{{Kind: posts.ContinentPlace, Continent: "South Asia", Tiles: 9600, Holder: "fr", Held: 6170, Share: 0.643, Changed: 6000, FlippedByContinent: map[string]int{"South Asia": 5200}}},
			Sagas: []posts.Saga{{
				Kind: posts.CountryPlace, Code: "cn", Continent: "East Asia", Tiles: 18622, Holder: "de", Share: 0.98,
				Since: at.Add(-25 * 24 * time.Hour), Until: at,
				Reigns: []posts.Reign{
					{
						Flag: "cz", PushedAt: at.Add(-24*24*time.Hour - 7*time.Hour), Since: at.Add(-24 * 24 * time.Hour), Until: at.Add(-19 * 24 * time.Hour),
						Peak: 0.98, Share: 0.4, TookFrom: "kp", TookFromShare: 0.33, LostTo: "de",
					},
					{Flag: "de", PushedAt: at.Add(-19*24*time.Hour - time.Hour), Since: at.Add(-19 * 24 * time.Hour), Peak: 0.98, Share: 0.98, TookFrom: "cz", TookFromShare: 0.91},
				},
				Contenders: []posts.Contender{
					{Flag: "cz", Since: at.Add(-24*24*time.Hour - 7*time.Hour), Until: at.Add(-19 * 24 * time.Hour), Peak: 0.98},
					{Flag: "de", Since: at.Add(-19*24*time.Hour - time.Hour), Peak: 0.98, Share: 0.98},
				},
			}},
			Paths: []posts.Path{
				{Flag: "de", Marks: []posts.PathMark{
					{At: at.Add(-12 * time.Hour), TilesByContinent: map[string]int{"East Asia": 18250, "Europe": 2}},
					{At: at, TilesByContinent: map[string]int{"East Asia": 18250, "Europe": 6000}},
				}},
				{Flag: "cz", Marks: []posts.PathMark{{At: at, TilesByContinent: map[string]int{"Europe": 400}}}},
			},
			Names:          map[string]string{"fr": "France", "tr": "Turkey"},
			Continents:     map[string]string{"Europe": "eu", "Middle East": ""},
			ContinentTiles: map[string]int{"Europe": 40000, "East Asia": 50000, "Middle East": 10000},
			Anthems:        []string{"eu", "fr", "tr"},
		},
		Told: []posts.Telling{
			{Title: "BRITTANY IS INVADING AUSTRALIA", Key: posts.StoryKey{Flag: "xb", Continent: "Oceania"}, Countries: []string{"au", "xb"}, At: at.Add(-time.Hour)},
			{
				Title: "CZECHIA CONQUERS CHINA", Key: posts.StoryKey{Flag: "cz", Continent: "East Asia", Angle: posts.Reigning}, Countries: []string{"cn", "cz"},
				At: at.Add(-23 * 24 * time.Hour),
				Facts: posts.TellingFacts{
					Kind: "saga", Attacker: "cz", Rival: "kp", Countries: []string{"cn"}, Line: "IN 7 HOURS",
					Since: at.Add(-24*24*time.Hour - 7*time.Hour), Until: at.Add(-23*24*time.Hour - 2*time.Hour), ShareAtStart: 0.0512, ShareAtEnd: 0.9799,
				},
			},
		},
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
			`{"front":1,"tour":0,"place":0,"kind":"battle","attacker":"fr","rival":"tr","takenFrom":["tr"],"countries":["tr"],"continent":"","team":"","headline":"FRANCE VS TURKEY","teasers":["THE WAR FOR TURKEY"],"line":"The battle for Turkey","call":"PICK A SIDE","callFlags":["fr","tr"],"anthem":"fr","look":"flags","fighters":[2,1]},`+
			`{"front":0,"tour":1,"place":0,"kind":"","attacker":"","rival":"","takenFrom":[],"countries":[],"continent":"","team":"","headline":"FRANCE IS EVERYWHERE","teasers":["NOBODY CAN STOP FRANCE"],"line":"","call":"FIGHT BACK","callFlags":[],"anthem":"","look":"","fighters":[]},`+
			`{"front":0,"tour":0,"place":0,"saga":1,"since":"2026-10-07T19:31:00Z","kind":"","attacker":"de","rival":"cz","takenFrom":[],"countries":[],"continent":"","team":"","headline":"GERMANY TAKES CHINA IN ONE HOUR","teasers":["CHINA FALLS TO GERMANY"],"line":"","call":"TAKE CHINA BACK","callFlags":["cz"],"anthem":"","look":"","fighters":[]}]}`)}

	edition, spend, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.NoError(t, err)
	assert.Equal(t, posts.Edition{Why: "The war is the day's story.", Stories: []posts.Story{
		{
			Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY", Teasers: []string{"THE WAR FOR TURKEY"},
			Line: "The battle for Turkey", Call: "PICK A SIDE", CallFlags: []string{"fr", "tr"}, Anthem: "fr", Look: posts.PaintedFlags,
			Fighters: []posts.FighterNumber{2, 1},
		},
		{
			Tour: 1, TakenFrom: []string{}, Countries: []string{}, Headline: "FRANCE IS EVERYWHERE", Teasers: []string{"NOBODY CAN STOP FRANCE"},
			Call: "FIGHT BACK", CallFlags: []string{},
		},
		{
			Saga: 1, Since: time.Date(2026, 10, 7, 19, 31, 0, 0, time.UTC), Attacker: "de", Rival: "cz", TakenFrom: []string{}, Countries: []string{}, Headline: "GERMANY TAKES CHINA IN ONE HOUR",
			Teasers: []string{"CHINA FALLS TO GERMANY"}, Call: "TAKE CHINA BACK", CallFlags: []string{"cz"},
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
	assert.Equal(t, []any{"why", "stories", "lab"}, format["schema"].(map[string]any)["required"])
	properties := format["schema"].(map[string]any)["properties"].(map[string]any)
	story := properties["stories"].(map[string]any)["items"].(map[string]any)
	fields := []any{"front", "tour", "place", "saga", "since", "kind", "attacker", "rival", "takenFrom", "countries", "continent", "team", "headline", "teasers", "line", "call", "callFlags", "anthem", "look", "fighters"}
	assert.Equal(t, fields, story["required"])
	lab := properties["lab"].(map[string]any)["items"].(map[string]any)
	assert.Equal(t, slices.Concat(fields, []any{"hook", "language"}), lab["required"], "a lab story has its hook and its language")
	assert.NotContains(t, story["properties"], "hook", "the control opens on no hook")

	prompt, err := claude_editor.PromptOf(offer)
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": prompt}}}},
		request["messages"])
	assert.Equal(t, key, api.headers[0].Get("X-Api-Key"))
}

func TestALabStoryIsReadWithItsHookAndItsLanguage(t *testing.T) {
	api := &fake{status: http.StatusOK, answer: message("end_turn", `{"why":"","stories":[],"lab":[`+
		`{"front":1,"tour":0,"place":0,"saga":0,"since":"","kind":"rout","attacker":"fr","rival":"","takenFrom":["tr"],"countries":["tr"],"continent":"","team":"",`+
		`"headline":"TÜRKİYE TÜRKİYE'Yİ KAYBEDİYOR","teasers":["FRANSA TÜRKİYE'Yİ ALIYOR"],"line":"","call":"TÜRKİYE'Yİ SAVUN","callFlags":["tr"],"anthem":"tr",`+
		`"look":"dive","fighters":[],"hook":"TÜRKLER, NEREDESİNİZ?","language":"tr"}]}`)}

	edition, _, err := editorOf(t, api.serve(t)).Edit(t.Context(), offer)

	require.NoError(t, err)
	assert.Empty(t, edition.Stories)
	require.Len(t, edition.Lab, 1)
	assert.Equal(t, "TÜRKLER, NEREDESİNİZ?", edition.Lab[0].Hook)
	assert.Equal(t, "tr", edition.Lab[0].Language)
	assert.Equal(t, posts.Rout, edition.Lab[0].Kind)
}

func TestThePromptGivesTheLabsBriefAndPickOnlyWhenTheRunHasOne(t *testing.T) {
	plain, err := claude_editor.PromptOf(offer)
	require.NoError(t, err)
	assert.NotContains(t, plain, `"lab"`)

	labbed := offer
	labbed.Lab = posts.LabSlot{At: 2, Experiment: posts.Experiment{Name: "far-invasion", Brief: "A flag far from home.", Format: posts.GlobeReplay}}
	prompt, err := claude_editor.PromptOf(labbed)
	require.NoError(t, err)

	var written map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(prompt, "The run's findings, as JSON:\n\n")), &written))
	assert.Equal(t, map[string]any{
		"experiment": "far-invasion", "pick": float64(2), "format": "globe-replay", "brief": "A flag far from home.",
	}, written["lab"])
	assert.NotContains(t, written, "lists")
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

func TestThePromptHoldsTheFindingsNumberedTheStoriesToldFirstAndThePostsTold(t *testing.T) {
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
				Code               string         `json:"code"`
				TookFrom           map[string]int `json:"tookFrom"`
				Solidity           float64        `json:"solidity"`
				FlippedByContinent map[string]int `json:"flippedByContinent"`
				Fighters           []struct {
					Number int    `json:"number"`
					Name   string `json:"name"`
					Took   int    `json:"took"`
				} `json:"fighters"`
			} `json:"flags"`
		} `json:"fronts"`
		Tours []struct {
			Tour               int            `json:"tour"`
			Flag               string         `json:"flag"`
			NextTook           int            `json:"nextTook"`
			FlippedByContinent map[string]int `json:"flippedByContinent"`
		} `json:"tours"`
		Places []struct {
			Place              int            `json:"place"`
			Kind               string         `json:"kind"`
			Continent          string         `json:"continent"`
			Share              float64        `json:"share"`
			FlippedByContinent map[string]int `json:"flippedByContinent"`
		} `json:"places"`
		Sagas       []json.RawMessage `json:"sagas"`
		Paths       json.RawMessage   `json:"paths"`
		Told        []json.RawMessage `json:"told"`
		ShownLately map[string]int    `json:"shownLately"`
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
	assert.Equal(t, map[string]int{"Middle East": 2400}, written.Fronts[0].Flags[0].FlippedByContinent)
	require.Len(t, written.Fronts[0].Flags[0].Fighters, 1)
	assert.Equal(t, "BraveFox42", written.Fronts[0].Flags[0].Fighters[0].Name)
	assert.Equal(t, 812, written.Fronts[0].Flags[0].Fighters[0].Took)
	assert.Equal(t, 1, written.Tours[0].Tour)
	assert.Equal(t, 9000, written.Tours[0].NextTook)
	assert.Equal(t, map[string]int{"South Asia": 6246}, written.Tours[0].FlippedByContinent)
	assert.Equal(t, 1, written.Places[0].Place)
	assert.Equal(t, "South Asia", written.Places[0].Continent)
	assert.Equal(t, map[string]int{"South Asia": 5200}, written.Places[0].FlippedByContinent)
	assert.Equal(t, map[string]int{"pw": 1}, written.ShownLately)
	require.Len(t, written.Sagas, 1)
	assert.JSONEq(t, `{"saga":1,"kind":"country","code":"cn","continent":"East Asia","tiles":18622,"holder":"de","share":0.98,`+
		`"since":"2026-10-01T20:31:00Z","until":"2026-10-26T20:31:00Z","reigns":[`+
		`{"reign":1,"flag":"cz","pushedAt":"2026-10-02T13:31:00Z","since":"2026-10-02T20:31:00Z","until":"2026-10-07T20:31:00Z","days":5,"conquestHours":7,"peak":0.98,"share":0.4,"tookFrom":"kp","tookFromShare":0.33,"lostTo":"de"},`+
		`{"reign":2,"flag":"de","pushedAt":"2026-10-07T19:31:00Z","since":"2026-10-07T20:31:00Z","days":19,"conquestHours":1,"peak":0.98,"share":0.98,"tookFrom":"cz","tookFromShare":0.91}],`+
		`"contenders":[{"flag":"cz","since":"2026-10-02T13:31:00Z","until":"2026-10-07T20:31:00Z","peak":0.98,"share":0},`+
		`{"flag":"de","since":"2026-10-07T19:31:00Z","peak":0.98,"share":0.98}]}`,
		string(written.Sagas[0]))
	assert.JSONEq(t, `{"at":["2026-10-26T08:31:00Z","2026-10-26T20:31:00Z"],"flags":[`+
		`{"flag":"de","percentOf":{"East Asia":[37,37],"Europe":[0,15],"the world":[18,24]}},`+
		`{"flag":"cz","percentOf":{"Europe":[0,1]}}]}`, string(written.Paths), "a continent never held 1% of is left out")
	require.Len(t, written.Told, 2)
	assert.JSONEq(t, `{"at":"2026-10-26T19:31:00Z","headline":"BRITTANY IS INVADING AUSTRALIA","key":{"flag":"xb","continent":"Oceania"},"shown":["au","xb"]}`,
		string(written.Told[0]), "a post from before its record kept what it told")
	assert.JSONEq(t, `{"at":"2026-10-03T20:31:00Z","headline":"CZECHIA CONQUERS CHINA","line":"IN 7 HOURS","key":{"flag":"cz","continent":"East Asia","angle":"saga"},`+
		`"kind":"saga","attacker":"cz","rival":"kp","countries":["cn"],"since":"2026-10-02T13:31:00Z","until":"2026-10-03T18:31:00Z",`+
		`"shareAtStart":0.051,"shareAtEnd":0.98}`,
		string(written.Told[1]))
}

func TestAPromptOver100KTokensIsBilledFiveTimesOver(t *testing.T) {
	short := claude_editor.SpendOf(claude_editor.DefaultModel, 100_000, 0, 0, 1000)
	long := claude_editor.SpendOf(claude_editor.DefaultModel, 100_001, 0, 0, 1000)

	assert.InDelta(t, 0.0105, short.USD, 1e-9)
	assert.InDelta(t, 5*(100_001*0.10+1000*0.50)/1e6, long.USD, 1e-9)
}

func carouselOffer() posts.Offer {
	listed := offer
	listed.Findings.Lists = posts.Lists{
		FightTheMost:   []posts.FightEntry{{Flag: "fr", Took: 30000, Lost: 11200, Rival: "pw", RivalTiles: 9000, Country: "fr"}},
		FarFromHome:    []posts.FarLandEntry{{Flag: "hu", Country: "id", Tiles: 2100, Km: 10400}},
		BiggestEmpires: []posts.EmpireEntry{{Flag: "pw", Tiles: 74000, Share: 0.28, Country: "ru"}},
		FoughtSince:    time.Date(2026, 10, 23, 20, 29, 0, 0, time.UTC),
	}
	listed.Told = []posts.Telling{{
		Key: posts.StoryKey{Flag: "fr", Continent: posts.TheWorld, Angle: posts.Listed}, Countries: []string{"fr", "tr"},
		At: time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC), Title: "THE 5 COUNTRIES THAT FIGHT THE MOST",
		List: posts.FightTheMost, Ranks: []string{"fr", "tr", "in", "de", "br"},
	}}
	listed.Lab = posts.LabSlot{At: 1, Experiment: posts.Experiment{Name: "carousel", Brief: "A list of countries.", Format: posts.Carousel}}
	return listed
}

func TestACarouselsPromptGivesTheListsAndEachCarouselToldWithItsListAndRanks(t *testing.T) {
	prompt, err := claude_editor.PromptOf(carouselOffer())
	require.NoError(t, err)

	var written map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(prompt, "The run's findings, as JSON:\n\n")), &written))
	assert.Equal(t, "carousel", written["lab"].(map[string]any)["format"])
	assert.Equal(t, map[string]any{
		"fight-the-most": []any{map[string]any{
			"flag": "fr", "took": float64(30000), "lost": float64(11200), "rival": "pw", "rivalTiles": float64(9000),
			"country": "fr",
		}},
		"far-from-home":   []any{map[string]any{"flag": "hu", "country": "id", "tiles": float64(2100), "km": float64(10400)}},
		"biggest-empires": []any{map[string]any{"flag": "pw", "tiles": float64(74000), "share": 0.28, "country": "ru"}},
		"foughtSince":     "2026-10-23T20:29:00Z",
	}, written["lists"])
	told := written["told"].([]any)[0].(map[string]any)
	assert.Equal(t, "fight-the-most", told["list"])
	assert.Equal(t, []any{"fr", "tr", "in", "de", "br"}, told["ranks"])
}

func TestACarouselsLabStoryIsAskedAsItsListItsCoverAndItsRanksAndReadSo(t *testing.T) {
	api := &fake{status: http.StatusOK, answer: message("end_turn", `{"why":"","stories":[],"lab":[`+
		`{"list":"fight-the-most","hook":"THE 5 COUNTRIES THAT FIGHT THE MOST","ranks":[{"code":"fr","line":"AGAINST PALAU"},{"code":"tr","line":"AGAINST FRANCE"}]}]}`)}

	edition, _, err := editorOf(t, api.serve(t)).Edit(t.Context(), carouselOffer())

	require.NoError(t, err)
	require.Len(t, edition.Lab, 1)
	assert.Equal(t, posts.Story{
		List: posts.FightTheMost, Hook: "THE 5 COUNTRIES THAT FIGHT THE MOST",
		Ranks: []posts.Rank{{Code: "fr", Line: "AGAINST PALAU"}, {Code: "tr", Line: "AGAINST FRANCE"}},
	}, edition.Lab[0])
	format := api.requests[0]["output_config"].(map[string]any)["format"].(map[string]any)
	lab := format["schema"].(map[string]any)["properties"].(map[string]any)["lab"].(map[string]any)["items"].(map[string]any)
	assert.Equal(t, []any{"list", "hook", "ranks"}, lab["required"])
	assert.Equal(t, []any{"fight-the-most", "far-from-home", "biggest-empires"},
		lab["properties"].(map[string]any)["list"].(map[string]any)["enum"])
}
