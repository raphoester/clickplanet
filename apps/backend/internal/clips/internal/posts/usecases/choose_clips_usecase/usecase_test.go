package choose_clips_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
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
	battle = posts.Story{Front: 1, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr"}, Headline: "FRANCE VS TURKEY", Teasers: []string{"THE WAR FOR TURKEY"}, Call: "PICK A SIDE", Look: posts.Dive}
)

type written struct {
	findings posts.Findings
}

func (w written) Findings(context.Context) (posts.Findings, error) {
	return w.findings, nil
}

type history struct {
	told  []posts.Telling
	err   error
	since []time.Time
	// How many posts each platform takes now, every one of the plan's budget when nil, and the platforms asked.
	left    map[posts.Platform]int
	roomErr error
	asked   [][]posts.Platform
}

func (h *history) Told(_ context.Context, since time.Time) ([]posts.Telling, error) {
	h.since = append(h.since, since)
	return h.told, h.err
}

func (h *history) Room(_ context.Context, platforms []posts.Platform) (map[posts.Platform]int, error) {
	h.asked = append(h.asked, platforms)
	if h.left == nil {
		return map[posts.Platform]int{posts.TikTok: 3, posts.YouTube: 12}, h.roomErr
	}
	return h.left, h.roomErr
}

var room = choose_clips_usecase.Room{PerRun: 2, Plan: posts.Plan{Platforms: []posts.Platform{posts.TikTok, posts.YouTube}}}

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

// A fake writer of the war report: what it answers, and each story it was asked to voice.
type narrator struct {
	script  narration.Script
	spend   posts.Spend
	err     error
	stories []posts.Story
	told    [][]posts.Telling
}

func (n *narrator) Narrate(_ context.Context, _ posts.Findings, story posts.Story, told []posts.Telling) (narration.Script, posts.Spend, error) {
	n.stories = append(n.stories, story)
	n.told = append(n.told, told)
	return n.script, n.spend, n.err
}

func editedBy(w written, h *history, model *editor) *choose_clips_usecase.UseCase {
	return choose_clips_usecase.New(w, h, h, cptime.NewFixedClock(now), room, model, &narrator{}, posts.LoadedDice(0))
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
	assert.Equal(t, past.told, model.offers[0].Told)
	assert.Equal(t, []time.Time{now.Add(-28 * 24 * time.Hour)}, past.since, "a season of posts told")
	assert.Equal(t, []posts.Chosen{
		{Pick: 1, Story: posts.Story{Seasonal: 1}, Key: day18},
		{Pick: 2, Story: battle, Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}},
	}, choice.Chosen)
	assert.Equal(t, posts.Editorial{Asked: true, Why: "the war", Spend: model.spend}, choice.Editorial)
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

func TestAHistoryThatCannotBeReadFailsTheRunRatherThanTellAStoryTwice(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")

	_, err := editedBy(written{findings: found}, &history{err: unreachable}, &editor{}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
}

func TestARunTakesNoMoreStoriesThanThePlatformsHaveRoomFor(t *testing.T) {
	past := &history{left: map[posts.Platform]int{posts.TikTok: 1, posts.YouTube: 0}}
	model := answering(battle)

	choice, err := editedBy(written{findings: found}, past, model).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []posts.Pick{1}, choice.Picks(), "the day won alone")
	assert.Empty(t, model.offers, "nothing left to decide")
	assert.Equal(t, 1, choice.Room)
	assert.Equal(t, [][]posts.Platform{room.Plan.Platforms}, past.asked)
}

func TestARunWhoseEveryPlatformPostedItsDayTakesNone(t *testing.T) {
	full := &history{left: map[posts.Platform]int{posts.TikTok: 0, posts.YouTube: 0}}

	choice, err := editedBy(written{findings: found}, full, answering(battle)).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, choice.Chosen)
	assert.Equal(t, 0, choice.Room)
}

func TestARoomThatCannotBeReadFailsTheRunRatherThanPostOverTheCap(t *testing.T) {
	unreachable := errors.New("ssh failed to ask GetRoom")

	_, err := editedBy(written{findings: found}, &history{roomErr: unreachable}, &editor{}).Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
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

func TestEachRunRollsTheLabsSlotBeforeTheEditorReadsAnythingAndGivesItItsBrief(t *testing.T) {
	far := posts.Experiment{Name: "far-invasion", Brief: "A flag far from home.", Format: posts.GlobeReplay}
	lab, err := posts.LabOf(far)
	require.NoError(t, err)
	labbed := room
	labbed.Lab = lab
	story := with(battle, "FRANCE STORMS TURKEY", "WHY IS FRANCE IN TURKEY?")
	model := &editor{edition: posts.Edition{Lab: []posts.Story{story}}, spend: posts.Spend{Model: "claude-haiku-5-5"}}

	// LoadedDice(2): the second pick, the first the day won takes.
	writer := &narrator{}
	choice, err := choose_clips_usecase.New(written{findings: found}, &history{}, &history{}, cptime.NewFixedClock(now), labbed, model, writer, posts.LoadedDice(2)).
		Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, model.offers, 1)
	assert.Equal(t, posts.LabSlot{At: 2, Experiment: far}, model.offers[0].Lab)
	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, posts.ExperimentName("far-invasion"), choice.Chosen[1].Story.Experiment)
	assert.True(t, choice.LabTold)
	assert.Empty(t, writer.stories, "no voice tells far invasion")
	assert.Equal(t, posts.Editorial{}, choice.Narration)
}

var (
	warReport = posts.Experiment{Name: "war-report", Brief: "One clear thread.", Format: posts.GlobeReplay, Narrated: true}
	turkey    = posts.Story{Front: 1, Kind: posts.Battle, Attacker: "tr", Rival: "fr", TakenFrom: []string{"fr"}, Countries: []string{"tr"}, Headline: "TURKEY VS FRANCE", Teasers: []string{"TURKEY FIGHTS BACK"}, Call: "PICK A SIDE", Look: posts.Dive}
)

// The war report at the second pick, the first the day won's; the control's story waits for room.
func reporting(t *testing.T, writer *narrator, told []posts.Telling) (posts.Choice, posts.Story) {
	t.Helper()
	lab, err := posts.LabOf(warReport)
	require.NoError(t, err)
	labbed := room
	labbed.Lab = lab
	story := battle
	story.Language = "en"
	model := &editor{edition: posts.Edition{Stories: []posts.Story{turkey}, Lab: []posts.Story{story}}, spend: posts.Spend{Model: "claude-haiku-5-5"}}

	choice, err := choose_clips_usecase.New(written{findings: found}, &history{told: told}, &history{}, cptime.NewFixedClock(now), labbed, model, writer, posts.LoadedDice(2)).
		Execute(t.Context())
	require.NoError(t, err)
	story.Experiment = warReport.Name
	return choice, story
}

func TestTheWarReportsStoryIsToldWithWhatItsVoiceSays(t *testing.T) {
	script := posts.Script{Title: "DAY 25: FRANCE STORMS TURKEY", Lines: []string{"Day twenty five.", "France is in Turkey."}}
	writer := &narrator{script: narration.Script{Script: script, Why: "the war"}, spend: posts.Spend{Model: "claude-sonnet-5-5", USD: 0.01}}
	episode := posts.Telling{Key: posts.StoryKey{Flag: "pw", Continent: "Europe"}, Title: "DAY 24", At: now.Add(-20 * time.Hour), Said: []string{"Day twenty four."}}

	choice, story := reporting(t, writer, []posts.Telling{episode})

	require.Len(t, writer.stories, 1)
	assert.Equal(t, posts.ExperimentName("war-report"), writer.stories[0].Experiment)
	assert.Equal(t, [][]posts.Telling{{episode}}, writer.told, "the season's posts, its war reports among them")
	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, script, choice.Chosen[1].Story.Script)
	assert.Equal(t, story.Headline, choice.Chosen[1].Story.Headline)
	assert.Empty(t, choice.Chosen[0].Story.Script, "the day won has no voice")
	assert.True(t, choice.LabTold)
	assert.Equal(t, posts.Editorial{Asked: true, Why: "the war", Spend: writer.spend}, choice.Narration)
}

func TestAWarReportTheWriterGivesNoVoiceLeavesItsPickToTheControlsNextStory(t *testing.T) {
	refused := errors.New("the model refused (general_harms)")
	writer := &narrator{err: refused, spend: posts.Spend{Model: "claude-sonnet-5-5", USD: 0.002}}

	choice, story := reporting(t, writer, nil)

	require.Len(t, choice.Chosen, 2)
	assert.Equal(t, posts.Story{Seasonal: 1}, choice.Chosen[0].Story)
	assert.Equal(t, turkey, choice.Chosen[1].Story)
	assert.False(t, choice.LabTold)
	assert.Equal(t, posts.Editorial{Asked: true, Spend: posts.Spend{Model: "claude-haiku-5-5"}}, choice.Editorial, "the editor's call is kept")
	assert.Equal(t, posts.Editorial{Asked: true, Failed: refused.Error(), Spend: writer.spend}, choice.Narration)
	assert.Contains(t, choice.Skipped, posts.Skipped{Story: story, Why: "the lab's war-report has no voice: " + refused.Error()})
}

func with(story posts.Story, headline, hook string) posts.Story {
	story.Headline, story.Hook = headline, hook
	return story
}
