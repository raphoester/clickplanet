package log_choose_clips_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase/log_choose_clips"
)

type stubExecutor struct {
	choice posts.Choice
	err    error
}

func (s stubExecutor) Execute(context.Context) (posts.Choice, error) {
	return s.choice, s.err
}

func run(t *testing.T, inner stubExecutor) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := log_choose_clips.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())
	return logs.String(), err
}

var (
	battle = posts.Story{Front: 3, Kind: posts.Battle, Attacker: "fr", Rival: "tr", TakenFrom: []string{"tr"}, Countries: []string{"tr", "sy"}, Headline: "FRANCE VS TURKEY", Teasers: []string{"THE WAR FOR TURKEY", "TURKEY FIGHTS BACK"}}
	day18  = posts.Chosen{Pick: 1, Story: posts.Story{Seasonal: 1}, Key: posts.StoryKey{Flag: "pw", Continent: "the world", Angle: posts.DayWon, Round: "0.18"}}
)

func TestEachChosenStoryIsLoggedWithWhatItTellsItsPickAndItsKey(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{Chosen: []posts.Chosen{
		day18,
		{Pick: 2, Story: battle, Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}},
		{Pick: 3, Story: posts.Story{Place: 2, Headline: "FRANCE RULES SOUTH ASIA"}, Key: posts.StoryKey{Flag: "fr", Continent: "South Asia", Angle: posts.Holding}},
	}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="chose a story" of="seasonal 1" pick=1 flag=pw continent="the world" angle=round round=0.18`)
	assert.Contains(t, logs, `msg="chose a story" of="front 3" headline="FRANCE VS TURKEY" teasers="THE WAR FOR TURKEY / TURKEY FIGHTS BACK" kind=battle attacker=fr from=tr in=tr+sy rival=tr pick=2 flag=fr continent="Middle East"`)
	assert.Contains(t, logs, `msg="chose a story" of="place 2" headline="FRANCE RULES SOUTH ASIA" pick=3 flag=fr continent="South Asia" angle=holding`)
}

func TestEachStoryOfTheEditorsSkippedIsLoggedWithWhy(t *testing.T) {
	wrong := battle
	wrong.Continent, wrong.Countries, wrong.Team = "Middle East", nil, "Middle East"
	logs, err := run(t, stubExecutor{choice: posts.Choice{Chosen: []posts.Chosen{day18}, Skipped: []posts.Skipped{{Story: wrong, Why: "a team is a continent's flags striking back together"}}}})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=WARN msg="skipped a story of the editor's" of="front 3" headline="FRANCE VS TURKEY" teasers="THE WAR FOR TURKEY / TURKEY FIGHTS BACK" kind=battle attacker=fr from=tr in="Middle East" rival=tr team="Middle East" why="a team is a continent's flags striking back together"`)
}

func TestAQuietRunIsLoggedAsSuch(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{Room: 2}})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="no story is told this run"`)
}

func TestARunWithNoRoomLeftSaysSoRatherThanThatNothingHappened(t *testing.T) {
	logs, err := run(t, stubExecutor{})

	require.NoError(t, err)
	assert.Contains(t, logs, `msg="every platform has posted its day already"`)
	assert.NotContains(t, logs, "no story is told")
}

func TestAFailureSaysNothing(t *testing.T) {
	logs, err := run(t, stubExecutor{err: errors.New("the file is gone")})

	require.Error(t, err)
	assert.Empty(t, logs)
}

func TestEachChosenStoryIsLoggedWithTheArmsOfItsClip(t *testing.T) {
	logs, err := run(t, stubExecutor{choice: posts.Choice{
		Chosen:   []posts.Chosen{day18, {Pick: 2, Story: battle, Key: posts.StoryKey{Flag: "fr", Continent: "Middle East"}}},
		Assigned: []posts.Assignment{{Pick: 1, Variants: posts.Variants{"length": "short"}}, {Pick: 2, Variants: posts.Variants{}}},
	}})

	require.NoError(t, err)
	assert.Contains(t, logs, `round=0.18 arms="length short"`)
	assert.Contains(t, logs, `continent="Middle East"`+"\n")
}

func TestHowTheEditorsCallWentIsLoggedWithWhatItCost(t *testing.T) {
	spend := posts.Spend{Model: "claude-haiku-5-5", InputTokens: 4200, CacheWriteTokens: 1800, OutputTokens: 600, USD: 0.00095}
	chosen := []posts.Chosen{day18}

	asked, err := run(t, stubExecutor{choice: posts.Choice{Chosen: chosen, Editorial: posts.Editorial{Asked: true, Why: "the war", Spend: spend}}})
	require.NoError(t, err)
	failed, err := run(t, stubExecutor{choice: posts.Choice{Chosen: chosen, Editorial: posts.Editorial{Failed: "no ANTHROPIC_API_KEY"}}})
	require.NoError(t, err)
	undecided, err := run(t, stubExecutor{choice: posts.Choice{Chosen: chosen}})
	require.NoError(t, err)
	quiet, err := run(t, stubExecutor{choice: posts.Choice{Room: 2}})
	require.NoError(t, err)

	assert.Contains(t, asked, `msg="the editor made the run" model=claude-haiku-5-5 input_tokens=4200 `+
		`cache_write_tokens=1800 cache_read_tokens=0 output_tokens=600 usd=0.00095 why="the war"`)
	assert.Contains(t, failed, `level=WARN msg="the editor gave no answer, and the run is its seasonal stories alone" why="no ANTHROPIC_API_KEY"`)
	assert.Contains(t, undecided, `msg="the editor had nothing to decide"`)
	assert.NotContains(t, quiet, "the editor")
}
