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
	now             = time.Date(2026, 10, 14, 0, 57, 0, 0, time.UTC)
	algeriaInAfrica = posts.StoryKey{Flag: "dz", Continent: "Africa"}
	stories         = []posts.Story{
		{Title: "ALGERIA STRIKES BACK", Key: algeriaInAfrica},
		{Title: "ALGERIA VS FRANCE", Key: algeriaInAfrica},
		{Title: "FRANCE IS ATTACKING EUROPE", Key: posts.StoryKey{Flag: "fr", Continent: "Europe"}},
		{Title: "PORTUGAL STRIKES BACK", Key: posts.StoryKey{Flag: "pt", Continent: "Europe"}},
	}
)

func shortlistOf(t *testing.T) posts.Shortlist {
	t.Helper()

	shortlist, err := posts.ShortlistOf(stories...)
	require.NoError(t, err)
	return shortlist
}

type written struct {
	shortlist posts.Shortlist
	err       error
}

func (w written) Shortlist(context.Context) (posts.Shortlist, error) {
	return w.shortlist, w.err
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
	PerRun: 2, Budget: posts.Budget{posts.TikTok: 10, posts.YouTube: 8}, Platforms: []posts.Platform{posts.TikTok, posts.YouTube},
}

func choosing(w written, h *history) *choose_clips_usecase.UseCase {
	return choose_clips_usecase.New(w, h, h, cptime.NewFixedClock(now), room, nil, posts.LoadedDice(0), &editor{})
}

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

func TestARunTakesTheBestStoriesNotToldLately(t *testing.T) {
	past := &history{told: []posts.Telling{{Key: algeriaInAfrica, At: now.Add(-4 * time.Hour)}}}

	choice, err := choosing(written{shortlist: shortlistOf(t)}, past).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{3, 4}, choice.Chosen.Picks())
	assert.Equal(t, []posts.Pick{1, 2}, choice.Retold.Picks())
	assert.Equal(t, []time.Time{now.Add(-7 * 24 * time.Hour)}, past.since)
}

func TestARunWithNoStoryWorthAClipTakesNone(t *testing.T) {
	choice, err := choosing(written{}, &history{}).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
}

func TestAShortlistThatCannotBeReadFailsTheRun(t *testing.T) {
	unreadable := errors.New("the file is gone")

	_, err := choosing(written{err: unreadable}, &history{}).Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
}

func TestAHistoryThatCannotBeReadFailsTheRunRatherThanTellAStoryTwice(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := choosing(written{shortlist: shortlistOf(t)}, &history{err: unreachable}).Execute(t.Context())

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

	choice, err := choosing(written{shortlist: shortlistOf(t)}, past).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1}, choice.Chosen.Picks())
	assert.Equal(t, 1, choice.Room)
	assert.Equal(t, []time.Time{now.Add(-24 * time.Hour)}, past.postedFrom)
}

func TestARunWhoseEveryPlatformPostedItsDayTakesNone(t *testing.T) {
	choice, err := choosing(written{shortlist: shortlistOf(t)}, &history{feed: hourly(10)}).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
	assert.Equal(t, 0, choice.Room)
}

func TestAFeedThatCannotBeReadFailsTheRunRatherThanPostOverTheCap(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := choosing(written{shortlist: shortlistOf(t)}, &history{feedErr: unreachable}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}

func TestEachStoryChosenIsPutInAnArmOfEveryExperimentAndNoneOtherIs(t *testing.T) {
	length := posts.Experiments{{Name: "length", Arms: posts.EvenArms("as-chosen", "short"), Control: "as-chosen"}}
	past := &history{}

	choice, err := choose_clips_usecase.New(
		written{shortlist: shortlistOf(t)}, past, past, cptime.NewFixedClock(now), room, length, &posts.RolledDice{Faces: []int{1, 0}},
		&editor{},
	).Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, []posts.Assignment{
		{Pick: choice.Chosen[0].Pick, Variants: posts.Variants{"length": "short"}},
		{Pick: choice.Chosen[1].Pick, Variants: posts.Variants{"length": "as-chosen"}},
	}, choice.Assigned)
}

func TestWithNoExperimentEachStoryChosenIsInNoArm(t *testing.T) {
	choice, err := choosing(written{shortlist: shortlistOf(t)}, &history{}).Execute(t.Context())

	require.NoError(t, err)
	require.NotEmpty(t, choice.Assigned)
	for _, assignment := range choice.Assigned {
		assert.Equal(t, posts.Variants{}, assignment.Variants)
	}
}

var (
	editing = posts.Experiments{{
		Name:    posts.EditorExperiment,
		Arms:    []posts.Arm{{Name: posts.ByRules, Weight: 1}, {Name: posts.ByModel, Weight: 1}, {Name: posts.FellBack}},
		Control: posts.ByRules, PerRun: true,
	}}
	// The run's coin first: 0 is the rules, 1 the model.
	byRules = posts.LoadedDice(0)
	byModel = posts.LoadedDice(1)
)

func editedBy(w written, h *history, dice posts.Dice, model *editor) *choose_clips_usecase.UseCase {
	return choose_clips_usecase.New(w, h, h, cptime.NewFixedClock(now), room, editing, dice, model)
}

func TestARunDrawnForTheRulesAsksNoModelAndIsWhatItWasBefore(t *testing.T) {
	model := &editor{}
	past := &history{told: []posts.Telling{{Key: algeriaInAfrica, At: now.Add(-4 * time.Hour)}}}

	choice, err := editedBy(written{shortlist: shortlistOf(t)}, past, byRules, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, model.offers)
	assert.Equal(t, []posts.Pick{3, 4}, choice.Chosen.Picks())
	assert.Equal(t, posts.Editorial{Arm: posts.ByRules}, choice.Editorial)
	assert.Equal(t, []posts.Assignment{
		{Pick: 3, Variants: posts.Variants{"editor": "rules"}},
		{Pick: 4, Variants: posts.Variants{"editor": "rules"}},
	}, choice.Assigned)
}

func TestARunDrawnForTheModelTellsItsStoriesWithItsHeadlines(t *testing.T) {
	model := &editor{
		edition: posts.Edition{Why: "Portugal is fresher.", Edits: []posts.Edit{
			{Pick: 4, Headline: "PORTUGAL TAKES EUROPE BACK"}, {Pick: 2, Headline: "ALGERIA AND FRANCE AT WAR"},
		}},
		spend: posts.Spend{Model: "claude-haiku-5-5", InputTokens: 3000, OutputTokens: 400, USD: 0.0005},
	}

	choice, err := editedBy(written{shortlist: shortlistOf(t)}, &history{}, byModel, model).Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, model.offers, 1)
	assert.Equal(t, []posts.Pick{1, 2, 3, 4}, model.offers[0].Stories.Picks())
	assert.Equal(t, []posts.Pick{2, 4}, choice.Chosen.Picks())
	assert.Equal(t, posts.Editorial{Arm: posts.ByModel, Asked: true, Why: "Portugal is fresher.", Spend: model.spend}, choice.Editorial)
	assert.Equal(t, []posts.Assignment{
		{Pick: 2, Variants: posts.Variants{"editor": "model"}, Headline: "ALGERIA AND FRANCE AT WAR"},
		{Pick: 4, Variants: posts.Variants{"editor": "model"}, Headline: "PORTUGAL TAKES EUROPE BACK"},
	}, choice.Assigned)
}

func TestAModelThatCannotAnswerLeavesTheRunToTheRulesAndSaysSo(t *testing.T) {
	refused := errors.New("the model refused (general_harms)")
	model := &editor{err: refused, spend: posts.Spend{Model: "claude-haiku-5-5", InputTokens: 3000, USD: 0.0003}}

	choice, err := editedBy(written{shortlist: shortlistOf(t)}, &history{}, byModel, model).Execute(t.Context())

	require.NoError(t, err, "never fails the run")
	assert.Equal(t, shortlistOf(t).Best(2, nil, now).Chosen.Picks(), choice.Chosen.Picks())
	assert.Equal(t, posts.Editorial{Arm: posts.ByModel, Asked: true, FellBack: refused.Error(), Spend: model.spend}, choice.Editorial)
	for _, assignment := range choice.Assigned {
		assert.Equal(t, posts.Variants{"editor": "fell-back"}, assignment.Variants)
		assert.Empty(t, assignment.Headline)
	}
}

func TestAnAnswerThatDoesNotHoldLeavesTheRunToTheRules(t *testing.T) {
	model := &editor{edition: posts.Edition{Edits: []posts.Edit{{Pick: 7, Headline: "A STORY NOBODY OFFERED"}}}, spend: posts.Spend{Model: "m"}}

	choice, err := editedBy(written{shortlist: shortlistOf(t)}, &history{}, byModel, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, shortlistOf(t).Best(2, nil, now).Chosen.Picks(), choice.Chosen.Picks())
	assert.Contains(t, choice.Editorial.FellBack, posts.ErrInvalidEdition.Error())
	assert.Equal(t, posts.Variants{"editor": "fell-back"}, choice.Assigned[0].Variants)
}

func TestWithNoKeyTheModelsRunFallsBackAndNothingWasAsked(t *testing.T) {
	model := &editor{err: errors.New("no ANTHROPIC_API_KEY")}

	choice, err := editedBy(written{shortlist: shortlistOf(t)}, &history{}, byModel, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Editorial{Arm: posts.ByModel, FellBack: "no ANTHROPIC_API_KEY"}, choice.Editorial)
	assert.Equal(t, posts.Variants{"editor": "fell-back"}, choice.Assigned[0].Variants)
}

func TestAQuietRunDrawnForTheModelAsksNobody(t *testing.T) {
	model := &editor{}

	choice, err := editedBy(written{}, &history{}, byModel, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, model.offers)
	assert.Equal(t, posts.Editorial{Arm: posts.ByModel}, choice.Editorial)
}
