package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func rush() cprules.Snapshot[*bonusesv1.Rules] {
	return cprules.Snapshot[*bonusesv1.Rules]{Name: "rush", Rules: &bonusesv1.Rules{Boxes: &bonusesv1.BoxPace{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: proto.Uint32(30),
	}}}
}

func restored(t *testing.T, snapshot cprules.Snapshot[*bonusesv1.Rules]) cprules.Set[*bonusesv1.Rules] {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := cprules.Snapshot[*bonusesv1.Rules]{Name: "quiet", Rules: &bonusesv1.Rules{
		Quiz:         &bonusesv1.QuizRules{Enabled: proto.Bool(false)},
		SignInRefill: proto.Bool(false),
	}}

	played := restored(t, quiet).TakePlayedSnapshot().Rules
	assert.False(t, played.GetQuiz().GetEnabled())
	assert.False(t, played.GetSignInRefill())
}

func TestADefaultThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	d := durationpb.New
	broken := map[string]func(*bonusesv1.Rules){
		"boxes.minInterval":                     func(r *bonusesv1.Rules) { r.Boxes.MinInterval = d(0) },
		"boxes.maxInterval":                     func(r *bonusesv1.Rules) { r.Boxes.MaxInterval = d(time.Minute) },
		"boxes.maxChargesPerHour":               func(r *bonusesv1.Rules) { r.Boxes.MaxChargesPerHour = proto.Uint32(0) },
		"kinds.bomb":                            func(r *bonusesv1.Rules) { r.Kinds.Bomb = proto.Float64(-1) },
		"kinds.refill is NaN":                   func(r *bonusesv1.Rules) { r.Kinds.Refill = proto.Float64(math.NaN()) },
		"kinds weigh every kind a box may be 0": func(r *bonusesv1.Rules) { r.Kinds = &bonusesv1.KindWeights{} },
		"offerTtl":                              func(r *bonusesv1.Rules) { r.OfferTtl = nil },
		"missRetry":                             func(r *bonusesv1.Rules) { r.MissRetry = d(0) },
		"activeWithin":                          func(r *bonusesv1.Rules) { r.ActiveWithin = d(0) },
		"spread.maxPerBox":                      func(r *bonusesv1.Rules) { r.Spread.MaxPerBox = proto.Uint32(0) },
		"enclose.maxPerBox":                     func(r *bonusesv1.Rules) { r.Enclose.MaxPerBox = proto.Uint32(0) },
		"shield.minPerBox":                      func(r *bonusesv1.Rules) { r.Shield.MinPerBox = proto.Uint32(0) },
		"shield.maxPerBox":                      func(r *bonusesv1.Rules) { r.Shield.MaxPerBox = proto.Uint32(4) },
		"welcome.within":                        func(r *bonusesv1.Rules) { r.Welcome.Within = d(-time.Minute) },
		"welcome.boxAfter":                      func(r *bonusesv1.Rules) { r.Welcome.BoxAfter = d(-time.Minute) },
		"quiz.minInterval":                      func(r *bonusesv1.Rules) { r.Quiz.MinInterval = d(0) },
		"quiz.maxInterval":                      func(r *bonusesv1.Rules) { r.Quiz.MaxInterval = d(time.Minute) },
		"quiz.maxChargesPerHour":                func(r *bonusesv1.Rules) { r.Quiz.MaxChargesPerHour = proto.Uint32(0) },
		"quiz.missRetry":                        func(r *bonusesv1.Rules) { r.Quiz.MissRetry = d(0) },
		"quiz.offerTtl":                         func(r *bonusesv1.Rules) { r.Quiz.OfferTtl = d(0) },
		"quiz.answerWindow":                     func(r *bonusesv1.Rules) { r.Quiz.AnswerWindow = d(0) },
		"quiz.leaderBias is -1":                 func(r *bonusesv1.Rules) { r.Quiz.LeaderBias = proto.Float64(-1) },
		"quiz.leaderBias is NaN":                func(r *bonusesv1.Rules) { r.Quiz.LeaderBias = proto.Float64(math.NaN()) },
		"quiz.target is 0.2, and a chance of a right answer is between 0.34 and 0.99": func(r *bonusesv1.Rules) {
			r.Quiz.Target = proto.Float64(0.2)
		},
		"quiz.target is 1":   func(r *bonusesv1.Rules) { r.Quiz.Target = proto.Float64(1) },
		"quiz.width is -1":   func(r *bonusesv1.Rules) { r.Quiz.Width = proto.Float64(-1) },
		"quiz.width is +Inf": func(r *bonusesv1.Rules) { r.Quiz.Width = proto.Float64(math.Inf(1)) },
		"quiz.floor is 1.5, and a share is between 0 and 1": func(r *bonusesv1.Rules) {
			r.Quiz.Floor = proto.Float64(1.5)
		},
		"quiz.floor is 0.8, over quiz.target (0.7)": func(r *bonusesv1.Rules) { r.Quiz.Floor = proto.Float64(0.8) },
		"quiz.floorAfter is 0, and must be above 0": func(r *bonusesv1.Rules) { r.Quiz.FloorAfter = proto.Uint32(0) },
		"quiz.explore is -0.1":                      func(r *bonusesv1.Rules) { r.Quiz.Explore = proto.Float64(-0.1) },
		"quiz.explore is 2":                         func(r *bonusesv1.Rules) { r.Quiz.Explore = proto.Float64(2) },
	}
	for reason, breaks := range broken {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named(cprules.Default, breaks).Restore(parts.All(), rules.Seeded())

			require.ErrorIs(t, err, cprules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestASetIsCheckedOverDefault(t *testing.T) {
	slower := cprules.Snapshot[*bonusesv1.Rules]{Name: "slower", Rules: &bonusesv1.Rules{
		Boxes: &bonusesv1.BoxPace{MinInterval: durationpb.New(10 * time.Minute)},
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "boxes.maxInterval is 8m0s, under boxes.minInterval (10m0s)")
}

func TestAFloorIsCheckedAgainstDefaultsTarget(t *testing.T) {
	harsh := cprules.Snapshot[*bonusesv1.Rules]{Name: "harsh", Rules: &bonusesv1.Rules{
		Quiz: &bonusesv1.QuizRules{Floor: proto.Float64(0.75)},
	}}

	_, err := harsh.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "quiz.floor is 0.75, over quiz.target (0.7)")
}

func TestTheDifficultyWeightedDrawIsOffInDefault(t *testing.T) {
	played := restored(t, rules.Seeded()).TakePlayedSnapshot().Rules.GetQuiz()

	assert.Zero(t, played.GetWidth(), "width 0 is the draw of before; the operator switches it on")
	assert.Equal(t, uint32(15), played.GetFloorAfter())
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *bonusesv1.Rules) { r.Boxes = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "boxes.minInterval is 0s")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *bonusesv1.Rules) {
		r.Spread.MaxPerBox = proto.Uint32(0)
		r.Quiz.AnswerWindow = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "spread.maxPerBox is 0")
	assert.Contains(t, err.Error(), "quiz.answerWindow is 0s")
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(cprules.Default, func(r *bonusesv1.Rules) {
		r.Kinds.Bomb = proto.Float64(0)
		r.Welcome = nil
		r.Quiz.LeaderBias = proto.Float64(0)
		r.Quiz.Enabled = proto.Bool(false)
		r.Quiz.Width = proto.Float64(0)
		r.Quiz.Floor = proto.Float64(0)
		r.Quiz.Explore = proto.Float64(0)
		r.Boxes.MaxInterval = r.Boxes.MinInterval
		r.SignInRefill = nil
		r.Frozen = nil
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}

func TestTheSetSeededIsOneThatPlays(t *testing.T) {
	_, err := rules.Seeded().Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}
