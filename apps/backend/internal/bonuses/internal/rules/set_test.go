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
)

func quicker(r *bonusesv1.Rules) {
	r.Boxes = &bonusesv1.BoxPace{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: proto.Uint32(30),
	}
}

func rush() rules.Snapshot {
	kept := rules.Snapshot{Name: "rush", Rules: &bonusesv1.Rules{}}
	quicker(kept.Rules)
	return kept
}

func restored(t *testing.T, snapshot rules.Snapshot) rules.Set {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestASetIsWhatItWasMadeOf(t *testing.T) {
	set := restored(t, rush())

	kept := set.TakeSnapshot()
	assert.Equal(t, rules.SetName("rush"), kept.Name)
	assert.True(t, proto.Equal(rush().Rules, kept.Rules))
}

func TestASetKeepsNothingItsSnapshotsShare(t *testing.T) {
	snapshot := rush()
	set := restored(t, snapshot)

	snapshot.Rules.Boxes.MaxChargesPerHour = proto.Uint32(9)
	set.TakeSnapshot().Rules.Boxes.MaxChargesPerHour = proto.Uint32(99)
	set.TakePlayedSnapshot().Rules.Shield.MaxPerBox = proto.Uint32(99)

	assert.True(t, set.Equal(restored(t, rush())))
}

func TestASetPlaysDefaultsValueWhereItHoldsNone(t *testing.T) {
	played := restored(t, rush()).TakePlayedSnapshot()

	assert.True(t, proto.Equal(rules.Named("rush", quicker).Rules, played.Rules), "%v", played.Rules)
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := rules.Snapshot{Name: "quiet", Rules: &bonusesv1.Rules{
		Quiz:         &bonusesv1.QuizRules{Enabled: proto.Bool(false)},
		SignInRefill: proto.Bool(false),
	}}

	played := restored(t, quiet).TakePlayedSnapshot().Rules
	assert.False(t, played.GetQuiz().GetEnabled())
	assert.False(t, played.GetSignInRefill())
}

func TestTwoSetsAreEqualWhenTheyPlayAlikeUnderOneName(t *testing.T) {
	assert.True(t, restored(t, rush()).Equal(restored(t, rush())))
	assert.True(t, restored(t, rush()).Equal(restored(t, rules.Named("rush", quicker))))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Named("rush", rules.AsIs))))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Snapshot{Name: "calm", Rules: rush().Rules})))
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
	}
	for reason, breaks := range broken {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named(rules.Default, breaks).Restore(parts.All(), rules.Seeded())

			require.ErrorIs(t, err, rules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestASetIsCheckedOverDefault(t *testing.T) {
	slower := rules.Snapshot{Name: "slower", Rules: &bonusesv1.Rules{
		Boxes: &bonusesv1.BoxPace{MinInterval: durationpb.New(10 * time.Minute)},
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "boxes.maxInterval is 8m0s, under boxes.minInterval (10m0s)")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *bonusesv1.Rules) { r.Boxes = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "boxes.minInterval is 0s")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *bonusesv1.Rules) {
		r.Spread.MaxPerBox = proto.Uint32(0)
		r.Quiz.AnswerWindow = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "spread.maxPerBox is 0")
	assert.Contains(t, err.Error(), "quiz.answerWindow is 0s")
}

func TestANameThatIsNotALowercaseWordIsRefused(t *testing.T) {
	for _, name := range []rules.SetName{"", "Rush", "a234567890123456789012345678901234"} {
		_, err := rules.Named(name, rules.AsIs).Restore(parts.All(), rules.Seeded())
		require.ErrorIs(t, err, rules.ErrInvalidSet, name)
	}
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(rules.Default, func(r *bonusesv1.Rules) {
		r.Kinds.Bomb = proto.Float64(0)
		r.Welcome = nil
		r.Quiz.LeaderBias = proto.Float64(0)
		r.Quiz.Enabled = proto.Bool(false)
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
