package rules_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
)

func quicker(r *planetv1.Rules) {
	if r.Clicks == nil {
		r.Clicks = &planetv1.ClickRules{}
	}
	if r.Bonuses == nil {
		r.Bonuses = &planetv1.BonusRules{}
	}
	r.Clicks.Tempo = proto.Float64(3)
	r.Bonuses.Boxes = &planetv1.BoxPace{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: proto.Uint32(30),
	}
}

func rush() rules.Snapshot {
	kept := rules.Snapshot{Name: "rush", Rules: &planetv1.Rules{}}
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

	snapshot.Rules.Clicks.Tempo = proto.Float64(9)
	set.TakeSnapshot().Rules.Clicks.Tempo = proto.Float64(99)
	set.TakePlayedSnapshot().Rules.Bonuses.Shield.PerTile = proto.Uint32(99)

	assert.True(t, set.Equal(restored(t, rush())))
}

func TestASetPlaysDefaultsValueWhereItHoldsNone(t *testing.T) {
	played := restored(t, rush()).TakePlayedSnapshot()

	assert.True(t, proto.Equal(rules.Named("rush", quicker).Rules, played.Rules), "%v", played.Rules)
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := rules.Snapshot{Name: "quiet", Rules: &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Quiz: &planetv1.QuizRules{Enabled: proto.Bool(false)}},
		Toll:    &planetv1.TollRules{},
	}}

	played := restored(t, quiet).TakePlayedSnapshot().Rules
	assert.False(t, played.GetBonuses().GetQuiz().GetEnabled())
	assert.Empty(t, played.GetToll().GetSteps())
}

func TestTwoSetsAreEqualWhenTheyPlayAlikeUnderOneName(t *testing.T) {
	assert.True(t, restored(t, rush()).Equal(restored(t, rush())))
	assert.True(t, restored(t, rush()).Equal(restored(t, rules.Named("rush", quicker))))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Named("rush", rules.AsIs))))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Snapshot{Name: "calm", Rules: rush().Rules})))
}

func TestADefaultThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	d := durationpb.New
	broken := map[string]func(*planetv1.Rules){
		"clicks.tempo":                     func(r *planetv1.Rules) { r.Clicks.Tempo = proto.Float64(0) },
		"clicks.perSecond":                 func(r *planetv1.Rules) { r.Clicks.PerSecond = proto.Float64(math.Inf(1)) },
		"clicks.burst":                     func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) },
		"clicks.scopeMultiplier":           func(r *planetv1.Rules) { r.Clicks.ScopeMultiplier = proto.Float64(0.5) },
		"clicks.v6ScopeMultiplier":         func(r *planetv1.Rules) { r.Clicks.V6ScopeMultiplier = proto.Float64(0) },
		"clicks.guestScopeMultiplier":      func(r *planetv1.Rules) { r.Clicks.GuestScopeMultiplier = proto.Float64(math.NaN()) },
		"clicks.linkedMultiplier":          func(r *planetv1.Rules) { r.Clicks.LinkedMultiplier = proto.Float64(0.5) },
		"clicks.newAccountClicks is -1":    func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = proto.Float64(-1) },
		"clicks.newAccountClicks is 61":    func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = proto.Float64(61) },
		"toll.steps[0].slowdown":           func(r *planetv1.Rules) { r.Toll.Steps[0].Slowdown = 1 },
		"toll.steps[1].slowdown":           func(r *planetv1.Rules) { r.Toll.Steps[1].Slowdown = 1.2 },
		"toll.steps[1].share":              func(r *planetv1.Rules) { r.Toll.Steps[1].Share = 0.05 },
		"toll.steps[2].share":              func(r *planetv1.Rules) { r.Toll.Steps[2].Share = 1.5 },
		"toll.steps[0].share":              func(r *planetv1.Rules) { r.Toll.Steps[0].Share = 0 },
		"bonuses.boxes.minInterval":        func(r *planetv1.Rules) { r.Bonuses.Boxes.MinInterval = d(0) },
		"bonuses.boxes.maxInterval":        func(r *planetv1.Rules) { r.Bonuses.Boxes.MaxInterval = d(time.Minute) },
		"bonuses.boxes.maxChargesPerHour":  func(r *planetv1.Rules) { r.Bonuses.Boxes.MaxChargesPerHour = proto.Uint32(0) },
		"bonuses.kinds.bomb":               func(r *planetv1.Rules) { r.Bonuses.Kinds.Bomb = proto.Float64(-1) },
		"bonuses.kinds weigh every kind 0": func(r *planetv1.Rules) { r.Bonuses.Kinds = &planetv1.KindWeights{} },
		"bonuses.offerTtl":                 func(r *planetv1.Rules) { r.Bonuses.OfferTtl = nil },
		"bonuses.missRetry":                func(r *planetv1.Rules) { r.Bonuses.MissRetry = d(0) },
		"bonuses.activeWithin":             func(r *planetv1.Rules) { r.Bonuses.ActiveWithin = d(0) },
		"bonuses.spread.clicks":            func(r *planetv1.Rules) { r.Bonuses.Spread.Clicks = proto.Uint32(0) },
		"bonuses.spread.maxPerBox":         func(r *planetv1.Rules) { r.Bonuses.Spread.MaxPerBox = proto.Uint32(0) },
		"bonuses.bomb.rings":               func(r *planetv1.Rules) { r.Bonuses.Bomb = nil },
		"bonuses.enclose.maxTiles":         func(r *planetv1.Rules) { r.Bonuses.Enclose.MaxTiles = proto.Uint32(0) },
		"bonuses.enclose.held":             func(r *planetv1.Rules) { r.Bonuses.Enclose.Held = proto.Uint32(0) },
		"bonuses.enclose.maxPerBox":        func(r *planetv1.Rules) { r.Bonuses.Enclose.MaxPerBox = proto.Uint32(0) },
		"bonuses.shield.held":              func(r *planetv1.Rules) { r.Bonuses.Shield.Held = proto.Uint32(0) },
		"bonuses.shield.minPerBox":         func(r *planetv1.Rules) { r.Bonuses.Shield.MinPerBox = proto.Uint32(0) },
		"bonuses.shield.maxPerBox":         func(r *planetv1.Rules) { r.Bonuses.Shield.MaxPerBox = proto.Uint32(4) },
		"bonuses.shield.perTile is 0":      func(r *planetv1.Rules) { r.Bonuses.Shield.PerTile = proto.Uint32(0) },
		"bonuses.shield.perTile is 256":    func(r *planetv1.Rules) { r.Bonuses.Shield.PerTile = proto.Uint32(256) },
		"bonuses.welcome.within":           func(r *planetv1.Rules) { r.Bonuses.Welcome.Within = d(-time.Minute) },
		"bonuses.welcome.boxAfter":         func(r *planetv1.Rules) { r.Bonuses.Welcome.BoxAfter = d(-time.Minute) },
		"bonuses.quiz.minInterval":         func(r *planetv1.Rules) { r.Bonuses.Quiz.MinInterval = d(0) },
		"bonuses.quiz.maxInterval":         func(r *planetv1.Rules) { r.Bonuses.Quiz.MaxInterval = d(time.Minute) },
		"bonuses.quiz.maxChargesPerHour":   func(r *planetv1.Rules) { r.Bonuses.Quiz.MaxChargesPerHour = proto.Uint32(0) },
		"bonuses.quiz.missRetry":           func(r *planetv1.Rules) { r.Bonuses.Quiz.MissRetry = d(0) },
		"bonuses.quiz.offerTtl":            func(r *planetv1.Rules) { r.Bonuses.Quiz.OfferTtl = d(0) },
		"bonuses.quiz.answerWindow":        func(r *planetv1.Rules) { r.Bonuses.Quiz.AnswerWindow = d(0) },
		"bonuses.quiz.leaderBias is -1":    func(r *planetv1.Rules) { r.Bonuses.Quiz.LeaderBias = proto.Float64(-1) },
		"bonuses.quiz.leaderBias is NaN":   func(r *planetv1.Rules) { r.Bonuses.Quiz.LeaderBias = proto.Float64(math.NaN()) },
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
	slower := rules.Snapshot{Name: "slower", Rules: &planetv1.Rules{
		Bonuses: &planetv1.BonusRules{Boxes: &planetv1.BoxPace{MinInterval: durationpb.New(10 * time.Minute)}},
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "bonuses.boxes.maxInterval")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.tempo is 0")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *planetv1.Rules) {
		r.Clicks.Burst = proto.Uint32(0)
		r.Bonuses.Quiz.AnswerWindow = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst is 0")
	assert.Contains(t, err.Error(), "bonuses.quiz.answerWindow is 0s")
}

func TestANameThatIsNotALowercaseWordIsRefused(t *testing.T) {
	for _, name := range []rules.SetName{"", "Rush", "a234567890123456789012345678901234"} {
		_, err := rules.Named(name, rules.AsIs).Restore(parts.All(), rules.Seeded())
		require.ErrorIs(t, err, rules.ErrInvalidSet, name)
	}
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(rules.Default, func(r *planetv1.Rules) {
		r.Toll = nil
		r.Clicks.NewAccountClicks = proto.Float64(0)
		r.Bonuses.Kinds.Bomb = proto.Float64(0)
		r.Bonuses.Welcome = nil
		r.Bonuses.Quiz.LeaderBias = proto.Float64(0)
		r.Bonuses.Quiz.Enabled = proto.Bool(false)
		r.Bonuses.Boxes.MaxInterval = r.Bonuses.Boxes.MinInterval
		r.Bonuses.Shield.PerTile = proto.Uint32(255)
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}

func TestTheSetSeededIsOneThatPlays(t *testing.T) {
	_, err := rules.Seeded().Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}
