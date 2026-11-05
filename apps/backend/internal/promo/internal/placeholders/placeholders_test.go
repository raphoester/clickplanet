package placeholders_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

var played = &planetv1.Rules{
	Clicks: &planetv1.ClickRules{Tempo: proto.Float64(3), PerSecond: proto.Float64(0.2), Burst: proto.Uint32(60)},
	Bonuses: &planetv1.BonusRules{
		Boxes: &planetv1.BoxPace{
			MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(90 * time.Second),
			MaxChargesPerHour: proto.Uint32(30),
		},
		Quiz:     &planetv1.QuizRules{OfferTtl: durationpb.New(1500 * time.Millisecond), Enabled: proto.Bool(true)},
		Welcome:  &planetv1.WelcomeRules{Within: durationpb.New(2 * time.Hour)},
		OfferTtl: durationpb.New(0),
	},
	Frozen: proto.Bool(false),
}

var triple = placeholders.Battle{Times: 3, Points: map[uint32]uint32{1: 75, 2: 54, 3: 45, 4: 36}}

func TestRulePathsAreTheNumbersAndDurationsOfTheRules(t *testing.T) {
	paths := placeholders.RulePaths()

	for _, path := range []string{
		"clicks.tempo", "clicks.burst", "bonuses.boxes.minInterval", "bonuses.bomb.rings",
		"bonuses.shield.perTile", "bonuses.quiz.leaderBias", "bonuses.offerTtl",
	} {
		assert.Contains(t, paths, path)
	}
	for _, path := range []string{"frozen", "allegianceLocked", "bonuses.quiz.enabled", "toll.steps", "clicks", "bonuses.boxes"} {
		assert.NotContains(t, paths, path, "a boolean, a list or a message reads as no word")
	}
}

func TestBattlePathsAreItsTimesAndThePointsOfItsFirstThreePlaces(t *testing.T) {
	assert.Equal(t, []string{"battle.times", "battle.points.first", "battle.points.second", "battle.points.third"},
		placeholders.BattlePaths())
	for _, path := range placeholders.RulePaths() {
		assert.NotContains(t, placeholders.BattlePaths(), path, "a rule and the battle never share a path")
	}
}

func TestFillWritesTheValuesTheSetPlays(t *testing.T) {
	for words, want := range map[string]string{
		"Clicks refill {clicks.tempo}× faster":                                   "Clicks refill 3× faster",
		"A click every {clicks.perSecond} s":                                     "A click every 0.2 s",
		"A bank of {clicks.burst}":                                               "A bank of 60",
		"A box every {bonuses.boxes.minInterval} to {bonuses.boxes.maxInterval}": "A box every 1 min to 90 s",
		"Answer within {bonuses.quiz.offerTtl}":                                  "Answer within 1500 ms",
		"Newcomers for {bonuses.welcome.within}":                                 "Newcomers for 2 h",
		"Gone in {bonuses.offerTtl}":                                             "Gone in 0 s",
		"No placeholder at all":                                                  "No placeholder at all",
		"":                                                                       "",
	} {
		filled, err := placeholders.Fill(words, played, triple)
		require.NoError(t, err, words)
		assert.Equal(t, want, filled)
	}
}

func TestFillWritesTheBattlesPoints(t *testing.T) {
	words := "×{battle.times}: 1st place wins {battle.points.first} points, 2nd {battle.points.second}, 3rd {battle.points.third}"
	for _, each := range []struct {
		battle placeholders.Battle
		want   string
	}{
		{triple, "×3: 1st place wins 75 points, 2nd 54, 3rd 45"},
		{placeholders.Battle{Times: 2, Points: map[uint32]uint32{1: 50, 2: 36, 3: 30}}, "×2: 1st place wins 50 points, 2nd 36, 3rd 30"},
	} {
		filled, err := placeholders.Fill(words, played, each.battle)
		require.NoError(t, err)
		assert.Equal(t, each.want, filled)
	}
}

func TestAValueTheSetDoesNotPlayFillsNothing(t *testing.T) {
	_, err := placeholders.Fill("{bonuses.bomb.rings} rings", played, triple)

	require.ErrorIs(t, err, placeholders.ErrNoValue)
}

func TestARankTheBattleDoesNotScoreFillsNothing(t *testing.T) {
	_, err := placeholders.Fill("{battle.points.third}", played, placeholders.Battle{Times: 1, Points: map[uint32]uint32{1: 25}})

	require.ErrorIs(t, err, placeholders.ErrNoValue)
}

func TestCheckRefusesWhatNamesNoValue(t *testing.T) {
	for _, words := range []string{
		"{clicks.speed}× faster",
		"{frozen}",
		"{toll.steps}",
		"{bonuses.boxes}",
		"{}",
		"an { alone",
		"a } alone",
		"{{clicks.tempo}}",
		"{battle}",
		"{battle.points}",
		"{battle.points.fourth}",
		"{battle.rule_set}",
	} {
		require.ErrorIs(t, placeholders.Check(words), placeholders.ErrInvalidPlaceholder, words)
		_, err := placeholders.Fill(words, played, triple)
		require.ErrorIs(t, err, placeholders.ErrInvalidPlaceholder, words)
	}
	require.NoError(t, placeholders.Check("{clicks.tempo}× faster, {bonuses.boxes.minInterval} a box"))
	require.NoError(t, placeholders.Check("1st place wins {battle.points.first} points"))
}
