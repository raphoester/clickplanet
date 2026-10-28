package choose_clips_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 10, 26, 20, 31, 0, 0, time.UTC)
	day18 = posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}
	// France and Turkey at war over Turkey, and a day won.
	found = posts.Findings{
		Fronts: []posts.Front{{
			Countries: []posts.FrontCountry{{Code: "tr", Continent: "Middle East", HoldersStart: map[string]int{"tr": 2000}, Takers: map[string]int{"fr": 5000, "tr": 200}}},
			Flags: []posts.FrontFlag{
				{Code: "fr", Continent: "Europe", Took: 5000, Lost: 3000, TookFrom: map[string]int{"tr": 5000}},
				{Code: "tr", Continent: "Middle East", Took: 3000, Lost: 5000, TookFrom: map[string]int{"fr": 3000}},
			},
		}},
		Seasonal: []posts.Seasonal{{Title: "PALAU WINS DAY 18", Key: day18}},
	}
	battle = posts.Story{Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY", Teasers: []string{"THE WAR FOR TURKEY"}, Call: "PICK A SIDE"}
)

type written struct {
	findings posts.Findings
	err      error
}

func (w written) Findings(context.Context) (posts.Findings, error) {
	return w.findings, w.err
}

type history struct {
	told  []posts.Telling
	err   error
	since []time.Time
	// What each platform holds, and since when it was asked.
	feed       posts.Feed
	feedErr    error
	postedFrom []time.Time
}

func (h *history) Told(_ context.Context, since time.Time) ([]posts.Telling, error) {
	h.since = append(h.since, since)
	return h.told, h.err
}

func (h *history) Posted(_ context.Context, since time.Time) (posts.Feed, error) {
	h.postedFrom = append(h.postedFrom, since)
	return h.feed, h.feedErr
}

var room = choose_clips_usecase.Room{
	PerRun: 2, Budget: posts.Budget{posts.TikTok: {PerDay: 10}, posts.YouTube: {PerDay: 8}}, Platforms: []posts.Platform{posts.TikTok, posts.YouTube},
}

var errUnkeyed = errors.New("no ANTHROPIC_API_KEY")

// A fake model: what it answers, and each offer it was asked about. No test calls the real one.
type editor struct {
	edition posts.Edition
	spend   posts.Spend
	err     error
	offers  []posts.Offer
}

func (e *editor) Edit(_ context.Context, offer posts.Offer) (posts.Edition, posts.Spend, error) {
	e.offers = append(e.offers, offer)
	return e.edition, e.spend, e.err
}

func editedBy(w written, h *history, model *editor) *choose_clips_usecase.UseCase {
	return choose_clips_usecase.New(w, h, h, cptime.NewFixedClock(now), room, nil, posts.LoadedDice(0), model)
}

func answering(stories ...posts.Story) *editor {
	return &editor{
		edition: posts.Edition{Why: "the war", Stories: stories},
		spend:   posts.Spend{Model: "claude-haiku-5-5", InputTokens: 6000, OutputTokens: 900, USD: 0.001},
	}
}

func TestTheEditorIsOfferedTheFindingsAndWhatWasToldAndItsStoriesAreTheRun(t *testing.T) {
	model := answering(battle)
	past := &history{told: []posts.Telling{{Key: posts.StoryKey{Flag: "xb", Continent: "Oceania"}, Title: "BRITTANY IS INVADING AUSTRALIA", At: now.Add(-2 * time.Hour)}}}

	choice, err := editedBy(written{findings: found}, past, model).Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, model.offers, 1)
	assert.Equal(t, found, model.offers[0].Findings)
	assert.Equal(t, 1, model.offers[0].Slots(), "the day won takes a slot")
	assert.Equal(t, past.told, model.offers[0].Recent)
	assert.Equal(t, []time.Time{now.Add(-7 * 24 * time.Hour)}, past.since)
	assert.Equal(t, []posts.Chosen{
		{Pick: 1, Story: posts.Story{Seasonal: 1}, Key: day18},
		{Pick: 2, Story: battle, Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}},
	}, choice.Chosen)
	assert.Equal(t, posts.Editorial{Asked: true, Why: "the war", Spend: model.spend}, choice.Editorial)
	assert.Equal(t, []posts.Assignment{
		{Pick: 1, Story: posts.Story{Seasonal: 1}, Variants: posts.Variants{}},
		{Pick: 2, Story: battle, Variants: posts.Variants{}},
	}, choice.Assigned)
}

func TestAStoryOfTheEditorsThatDoesNotHoldIsSkippedAndTheRunGoesOn(t *testing.T) {
	wrong := battle
	wrong.Attacker = "pw"
	model := answering(wrong, battle)

	choice, err := editedBy(written{findings: found}, &history{}, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1, 2}, choice.Picks())
	assert.Equal(t, battle, choice.Chosen[1].Story)
	require.Len(t, choice.Skipped, 1)
	assert.Contains(t, choice.Skipped[0].Why, `"pw" took nothing`)
}

func TestARunWithNothingFoundTakesNoneAndAsksNobody(t *testing.T) {
	model := &editor{}

	choice, err := editedBy(written{}, &history{}, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
	assert.Empty(t, model.offers)
	assert.Equal(t, posts.Editorial{}, choice.Editorial)
}

func TestFindingsThatCannotBeReadFailTheRun(t *testing.T) {
	unreadable := errors.New("the file is gone")

	_, err := editedBy(written{err: unreadable}, &history{}, &editor{}).Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
}

func TestAHistoryThatCannotBeReadFailsTheRunRatherThanTellAStoryTwice(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := editedBy(written{findings: found}, &history{err: unreachable}, &editor{}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}

// count posts on every platform, an hour apart, the newest an hour ago.
func hourly(count int) posts.Feed {
	feed := make(posts.Feed, 0, count)
	for i := range count {
		feed = append(feed, posts.Posting{At: now.Add(-time.Duration(i+1) * time.Hour), Platforms: room.Platforms})
	}
	return feed
}

func TestARunTakesNoMoreStoriesThanThePlatformsHaveRoomFor(t *testing.T) {
	past := &history{feed: hourly(9)}
	model := answering(battle)

	choice, err := editedBy(written{findings: found}, past, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1}, choice.Picks(), "the day won alone")
	assert.Empty(t, model.offers, "nothing left to decide")
	assert.Equal(t, 1, choice.Room)
	assert.Equal(t, []time.Time{now.Add(-24 * time.Hour)}, past.postedFrom)
}

func TestARunWhoseEveryPlatformPostedItsDayTakesNone(t *testing.T) {
	choice, err := editedBy(written{findings: found}, &history{feed: hourly(10)}, answering(battle)).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
	assert.Equal(t, 0, choice.Room)
}

func TestAFeedThatCannotBeReadFailsTheRunRatherThanPostOverTheCap(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := editedBy(written{findings: found}, &history{feedErr: unreachable}, &editor{}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}

func TestEachStoryChosenIsPutInAnArmOfEveryExperiment(t *testing.T) {
	length := posts.Experiments{{Name: "length", Arms: posts.EvenArms("as-chosen", "short"), Control: "as-chosen"}}
	past := &history{}

	choice, err := choose_clips_usecase.New(
		written{findings: found}, past, past, cptime.NewFixedClock(now), room, length, &posts.RolledDice{Faces: []int{1, 0}},
		answering(battle),
	).Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, []posts.Assignment{
		{Pick: 1, Story: posts.Story{Seasonal: 1}, Variants: posts.Variants{"length": "short"}},
		{Pick: 2, Story: battle, Variants: posts.Variants{"length": "as-chosen"}},
	}, choice.Assigned)
}

func TestAnEditorThatCannotAnswerLeavesTheRunItsDayWonAndSaysWhy(t *testing.T) {
	refused := errors.New("the model refused (general_harms)")
	model := &editor{err: refused, spend: posts.Spend{Model: "claude-haiku-5-5", InputTokens: 3000, USD: 0.0003}}

	choice, err := editedBy(written{findings: found}, &history{}, model).Execute(t.Context())

	require.NoError(t, err, "never fails the run")
	assert.Equal(t, []posts.Pick{1}, choice.Picks())
	assert.Equal(t, posts.Story{Seasonal: 1}, choice.Chosen[0].Story)
	assert.Equal(t, posts.Editorial{Asked: true, Failed: refused.Error(), Spend: model.spend}, choice.Editorial)
}

func TestWithNoKeyNothingWasAskedAndTheRunIsItsDayWon(t *testing.T) {
	choice, err := editedBy(written{findings: found}, &history{}, &editor{err: errUnkeyed}).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1}, choice.Picks())
	assert.Equal(t, posts.Editorial{Failed: "no ANTHROPIC_API_KEY"}, choice.Editorial)
}

func TestADayWonToldAlreadyIsNotToldAgain(t *testing.T) {
	past := &history{told: []posts.Telling{{Key: day18, Title: "PALAU WINS DAY 18", At: now.Add(-2 * time.Hour)}}}

	choice, err := editedBy(written{findings: found}, past, answering(battle)).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Chosen{{Pick: 1, Story: battle, Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}}}, choice.Chosen)
}
