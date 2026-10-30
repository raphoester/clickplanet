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

func rush() rules.Snapshot {
	return rules.Named("rush", func(r *planetv1.Rules) {
		r.Clicks.Tempo = 3
		r.Bonuses.Boxes = &planetv1.BoxPace{
			MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: 30,
		}
	})
}

func restored(t *testing.T, snapshot rules.Snapshot) rules.Set {
	t.Helper()

	set, err := snapshot.Restore(parts.All())
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

	snapshot.Rules.Clicks.Tempo = 9
	set.TakeSnapshot().Rules.Bonuses.Shield.PerTile = 99

	assert.True(t, set.Equal(restored(t, rush())))
}

func TestTwoSetsAreEqualWhenTheyPlayAlikeUnderOneName(t *testing.T) {
	assert.True(t, restored(t, rush()).Equal(restored(t, rush())))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Named("rush", rules.AsIs))))
	assert.False(t, restored(t, rush()).Equal(restored(t, rules.Snapshot{Name: "calm", Rules: rush().Rules})))
}

func TestASetThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	d := durationpb.New
	broken := map[string]func(*planetv1.Rules){
		"clicks.tempo":                         func(r *planetv1.Rules) { r.Clicks.Tempo = 0 },
		"clicks.perSecond":                     func(r *planetv1.Rules) { r.Clicks.PerSecond = math.Inf(1) },
		"clicks.burst":                         func(r *planetv1.Rules) { r.Clicks.Burst = 0 },
		"clicks.scopeMultiplier":               func(r *planetv1.Rules) { r.Clicks.ScopeMultiplier = 0.5 },
		"clicks.v6ScopeMultiplier":             func(r *planetv1.Rules) { r.Clicks.V6ScopeMultiplier = 0 },
		"clicks.guestScopeMultiplier":          func(r *planetv1.Rules) { r.Clicks.GuestScopeMultiplier = math.NaN() },
		"clicks.linkedMultiplier":              func(r *planetv1.Rules) { r.Clicks.LinkedMultiplier = 0.5 },
		"clicks.newAccountClicks is -1":        func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = -1 },
		"clicks.newAccountClicks is 61":        func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = 61 },
		"toll.steps[0].slowdown":               func(r *planetv1.Rules) { r.Toll.Steps[0].Slowdown = 1 },
		"toll.steps[1].slowdown":               func(r *planetv1.Rules) { r.Toll.Steps[1].Slowdown = 1.2 },
		"toll.steps[1].share":                  func(r *planetv1.Rules) { r.Toll.Steps[1].Share = 0.05 },
		"toll.steps[2].share":                  func(r *planetv1.Rules) { r.Toll.Steps[2].Share = 1.5 },
		"toll.steps[0].share":                  func(r *planetv1.Rules) { r.Toll.Steps[0].Share = 0 },
		"bonuses.boxes.minInterval":            func(r *planetv1.Rules) { r.Bonuses.Boxes.MinInterval = d(0) },
		"bonuses.boxes.maxInterval":            func(r *planetv1.Rules) { r.Bonuses.Boxes.MaxInterval = d(time.Minute) },
		"bonuses.boxes.maxChargesPerHour":      func(r *planetv1.Rules) { r.Bonuses.Boxes.MaxChargesPerHour = 0 },
		"bonuses.kinds.bomb":                   func(r *planetv1.Rules) { r.Bonuses.Kinds.Bomb = -1 },
		"bonuses.kinds weigh every kind 0":     func(r *planetv1.Rules) { r.Bonuses.Kinds = &planetv1.KindWeights{} },
		"bonuses.offerTtl":                     func(r *planetv1.Rules) { r.Bonuses.OfferTtl = nil },
		"bonuses.missRetry":                    func(r *planetv1.Rules) { r.Bonuses.MissRetry = d(0) },
		"bonuses.activeWithin":                 func(r *planetv1.Rules) { r.Bonuses.ActiveWithin = d(0) },
		"bonuses.spread.clicks":                func(r *planetv1.Rules) { r.Bonuses.Spread.Clicks = 0 },
		"bonuses.spread.maxPerBox":             func(r *planetv1.Rules) { r.Bonuses.Spread.MaxPerBox = 0 },
		"bonuses.bomb.rings":                   func(r *planetv1.Rules) { r.Bonuses.Bomb = nil },
		"bonuses.enclose.maxTiles":             func(r *planetv1.Rules) { r.Bonuses.Enclose.MaxTiles = 0 },
		"bonuses.enclose.held":                 func(r *planetv1.Rules) { r.Bonuses.Enclose.Held = 0 },
		"bonuses.enclose.maxPerBox":            func(r *planetv1.Rules) { r.Bonuses.Enclose.MaxPerBox = 0 },
		"bonuses.shield.held":                  func(r *planetv1.Rules) { r.Bonuses.Shield.Held = 0 },
		"bonuses.shield.minPerBox":             func(r *planetv1.Rules) { r.Bonuses.Shield.MinPerBox = 0 },
		"bonuses.shield.maxPerBox":             func(r *planetv1.Rules) { r.Bonuses.Shield.MaxPerBox = 4 },
		"bonuses.shield.perTile is 0":          func(r *planetv1.Rules) { r.Bonuses.Shield.PerTile = 0 },
		"bonuses.shield.perTile is 256":        func(r *planetv1.Rules) { r.Bonuses.Shield.PerTile = 256 },
		"bonuses.welcome.within":               func(r *planetv1.Rules) { r.Bonuses.Welcome.Within = d(-time.Minute) },
		"bonuses.welcome.boxAfter":             func(r *planetv1.Rules) { r.Bonuses.Welcome.BoxAfter = d(-time.Minute) },
		"bonuses.quiz.minInterval":             func(r *planetv1.Rules) { r.Bonuses.Quiz.MinInterval = d(0) },
		"bonuses.quiz.maxInterval":             func(r *planetv1.Rules) { r.Bonuses.Quiz.MaxInterval = d(time.Minute) },
		"bonuses.quiz.maxChargesPerHour":       func(r *planetv1.Rules) { r.Bonuses.Quiz.MaxChargesPerHour = 0 },
		"bonuses.quiz.missRetry":               func(r *planetv1.Rules) { r.Bonuses.Quiz.MissRetry = d(0) },
		"bonuses.quiz.offerTtl":                func(r *planetv1.Rules) { r.Bonuses.Quiz.OfferTtl = d(0) },
		"bonuses.quiz.answerWindow":            func(r *planetv1.Rules) { r.Bonuses.Quiz.AnswerWindow = d(0) },
		"bonuses.quiz.leaderBias is -1":        func(r *planetv1.Rules) { r.Bonuses.Quiz.LeaderBias = -1 },
		"bonuses.quiz.leaderBias is NaN":       func(r *planetv1.Rules) { r.Bonuses.Quiz.LeaderBias = math.NaN() },
		"clicks.tempo is 0, and must be above": func(r *planetv1.Rules) { r.Clicks = nil },
	}
	for reason, breaks := range broken {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named("rush", breaks).Restore(parts.All())

			require.ErrorIs(t, err, rules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named("rush", func(r *planetv1.Rules) {
		r.Clicks.Burst = 0
		r.Bonuses.Quiz.AnswerWindow = nil
	}).Restore(parts.All())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst is 0")
	assert.Contains(t, err.Error(), "bonuses.quiz.answerWindow is 0s")
}

func TestANameThatIsNotALowercaseWordIsRefused(t *testing.T) {
	for _, name := range []rules.SetName{"", "Rush", "a234567890123456789012345678901234"} {
		_, err := rules.Named(name, rules.AsIs).Restore(parts.All())
		require.ErrorIs(t, err, rules.ErrInvalidSet, name)
	}
}

func TestEveryValueThatCanBeZeroIsASet(t *testing.T) {
	set := rules.Named("calm", func(r *planetv1.Rules) {
		r.Toll = nil
		r.Clicks.NewAccountClicks = 0
		r.Bonuses.Kinds.Bomb = 0
		r.Bonuses.Welcome = nil
		r.Bonuses.Quiz.LeaderBias = 0
		r.Bonuses.Quiz.Enabled = false
		r.Bonuses.Boxes.MaxInterval = r.Bonuses.Boxes.MinInterval
		r.Bonuses.Shield.PerTile = 255
	})

	_, err := set.Restore(parts.All())
	require.NoError(t, err)
}

func TestTheSetSeededIsOneThatPlays(t *testing.T) {
	_, err := rules.Seeded().Restore(parts.All())
	require.NoError(t, err)
}
