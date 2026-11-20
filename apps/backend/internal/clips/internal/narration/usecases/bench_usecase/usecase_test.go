package bench_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration/usecases/bench_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

var (
	until = time.Date(2026, 11, 20, 8, 24, 0, 0, time.UTC)
	found = posts.Findings{
		Until: until,
		Fronts: []posts.Front{{
			Countries: []posts.FrontCountry{{Code: "tr", Continent: "Middle East", HoldersStart: map[string]int{"tr": 2000}, Takers: map[string]int{"fr": 5000}}},
			Flags: []posts.FrontFlag{
				{Code: "fr", Continent: "Europe", Took: 5000, Lost: 3000, TookFrom: map[string]int{"tr": 5000}},
				{Code: "tr", Continent: "Middle East", Took: 3000, Lost: 5000, TookFrom: map[string]int{"fr": 3000}},
			},
		}},
		Seasonal: []posts.Seasonal{{Title: "PALAU WINS DAY 18", Key: posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}}},
	}
	invasion = posts.Story{
		Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr"},
		Headline: "FRANCE INVADES TURKEY", Teasers: []string{"TURKEY UNDER ATTACK"}, Call: "DEFEND TURKEY", Look: posts.Dive,
	}
	correspondent = narration.Tone{Name: "correspondent", Brief: "# War correspondent"}
	deadpan       = narration.Tone{Name: "deadpan", Brief: "# Deadpan"}
)

type written posts.Findings

func (w written) Findings(context.Context) (posts.Findings, error) {
	return posts.Findings(w), nil
}

type editor struct {
	stories []posts.Story
	offers  []posts.Offer
}

func (e *editor) Edit(_ context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error) {
	e.offers = append(e.offers, offer)
	return posts.Edition{Stories: e.stories, Why: "the war"}, posts.Spend{Model: "claude-haiku-5-5", USD: 0.002}, nil
}

type writer struct {
	requests []narration.Request
	models   []narration.Model
	refuse   narration.ToneName
}

var errRefused = errors.New("the model refused")

func (w *writer) Write(_ context.Context, model narration.Model, request narration.Request) (narration.Script, posts.Spend, error) {
	w.requests = append(w.requests, request)
	w.models = append(w.models, model)
	if request.Tone.Name == w.refuse {
		return narration.Script{}, posts.Spend{Model: string(model), USD: 0.01}, errRefused
	}
	return narration.Script{Title: "DAY 9", Lines: []string{string(request.Tone.Name)}}, posts.Spend{Model: string(model), USD: 0.05}, nil
}

func facts(posts.Findings, posts.Story) (string, error) {
	return `{"front":{"front":1}}`, nil
}

func TestEachStoryTheEditorChoseIsWrittenInEachToneByEachModel(t *testing.T) {
	model := &editor{stories: []posts.Story{invasion}}
	scripts := &writer{refuse: "deadpan"}
	round := bench_usecase.Round{Stories: 2, Tones: []narration.Tone{correspondent, deadpan}, Models: []narration.Model{narration.Opus, narration.Haiku}, Day: 9}

	bench, err := bench_usecase.New(written(found), model, facts, scripts, round).Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, model.offers, 1)
	assert.Equal(t, 2, model.offers[0].Slots(), "no day won takes a slot: the editor fills them all")
	assert.Equal(t, until, model.offers[0].Now, "the moment the findings measured, not the bench's")
	require.Len(t, bench.Stories, 1)
	assert.Equal(t, "FRANCE INVADES TURKEY", bench.Stories[0].Headline)
	require.Len(t, bench.Entries, 4)
	assert.Equal(t, []narration.Model{narration.Opus, narration.Haiku, narration.Opus, narration.Haiku}, scripts.models)
	assert.Equal(t, narration.Request{Story: bench.Stories[0], Facts: `{"front":{"front":1}}`, Tone: correspondent, Day: 9}, scripts.requests[0])
	assert.Equal(t, narration.Entry{Story: 1, Tone: "correspondent", Model: narration.Opus, Script: narration.Script{Title: "DAY 9", Lines: []string{"correspondent"}}, Spend: posts.Spend{Model: "claude-opus-5-5", USD: 0.05}}, bench.Entries[0])
	assert.Equal(t, "the model refused", bench.Entries[2].Failed, "one refusal leaves the others to compare")
	assert.InDelta(t, 0.002+0.05+0.05+0.01+0.01, bench.USD(), 1e-9)
	assert.Equal(t, "the war", bench.Editorial.Why)
}
