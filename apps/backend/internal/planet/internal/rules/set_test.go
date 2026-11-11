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
	if r.Charges == nil {
		r.Charges = &planetv1.ChargeRules{}
	}
	r.Clicks.Tempo = proto.Float64(3)
	r.Charges.Spread = &planetv1.SpreadRules{Clicks: proto.Uint32(30)}
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
	set.TakePlayedSnapshot().Rules.Charges.Shield.PerTile = proto.Uint32(99)

	assert.True(t, set.Equal(restored(t, rush())))
}

func TestASetPlaysDefaultsValueWhereItHoldsNone(t *testing.T) {
	played := restored(t, rush()).TakePlayedSnapshot()

	assert.True(t, proto.Equal(rules.Named("rush", quicker).Rules, played.Rules), "%v", played.Rules)
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := rules.Snapshot{Name: "quiet", Rules: &planetv1.Rules{
		Charges: &planetv1.ChargeRules{Bomb: &planetv1.BombRules{Off: proto.Bool(false)}},
		Toll:    &planetv1.TollRules{},
	}}

	played := restored(t, quiet).TakePlayedSnapshot().Rules
	assert.False(t, played.GetCharges().GetBomb().GetOff())
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
		"clicks.tempo":                  func(r *planetv1.Rules) { r.Clicks.Tempo = proto.Float64(0) },
		"clicks.perSecond":              func(r *planetv1.Rules) { r.Clicks.PerSecond = proto.Float64(math.Inf(1)) },
		"clicks.burst":                  func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) },
		"clicks.scopeMultiplier":        func(r *planetv1.Rules) { r.Clicks.ScopeMultiplier = proto.Float64(0.5) },
		"clicks.v6ScopeMultiplier":      func(r *planetv1.Rules) { r.Clicks.V6ScopeMultiplier = proto.Float64(0) },
		"clicks.guestScopeMultiplier":   func(r *planetv1.Rules) { r.Clicks.GuestScopeMultiplier = proto.Float64(math.NaN()) },
		"clicks.linkedMultiplier":       func(r *planetv1.Rules) { r.Clicks.LinkedMultiplier = proto.Float64(0.5) },
		"clicks.newAccountClicks is -1": func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = proto.Float64(-1) },
		"clicks.newAccountClicks is 61": func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = proto.Float64(61) },
		"clicks.fastStart":              func(r *planetv1.Rules) { r.Clicks.FastStart = d(-time.Minute) },
		"toll.steps[0].slowdown":        func(r *planetv1.Rules) { r.Toll.Steps[0].Slowdown = 1 },
		"toll.steps[1].slowdown":        func(r *planetv1.Rules) { r.Toll.Steps[1].Slowdown = 1.2 },
		"toll.steps[1].share":           func(r *planetv1.Rules) { r.Toll.Steps[1].Share = 0.05 },
		"toll.steps[2].share":           func(r *planetv1.Rules) { r.Toll.Steps[2].Share = 1.5 },
		"toll.steps[0].share":           func(r *planetv1.Rules) { r.Toll.Steps[0].Share = 0 },
		"charges.spread.clicks":         func(r *planetv1.Rules) { r.Charges.Spread.Clicks = proto.Uint32(0) },
		"charges.bomb.rings":            func(r *planetv1.Rules) { r.Charges.Bomb = nil },
		"charges.enclose.maxTiles":      func(r *planetv1.Rules) { r.Charges.Enclose.MaxTiles = proto.Uint32(0) },
		"charges.enclose.held":          func(r *planetv1.Rules) { r.Charges.Enclose.Held = proto.Uint32(0) },
		"charges.shield.held":           func(r *planetv1.Rules) { r.Charges.Shield.Held = proto.Uint32(0) },
		"charges.shield.perTile is 0":   func(r *planetv1.Rules) { r.Charges.Shield.PerTile = proto.Uint32(0) },
		"charges.shield.perTile is 256": func(r *planetv1.Rules) { r.Charges.Shield.PerTile = proto.Uint32(256) },
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
		Clicks: &planetv1.ClickRules{NewAccountClicks: proto.Float64(100)},
	}}

	_, err := slower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.newAccountClicks is 100")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.tempo is 0")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(rules.Default, func(r *planetv1.Rules) {
		r.Clicks.Burst = proto.Uint32(0)
		r.Charges.Enclose.Held = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.burst is 0")
	assert.Contains(t, err.Error(), "charges.enclose.held is 0")
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
		r.Charges.Shield.PerTile = proto.Uint32(255)
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}

func TestTheSetSeededIsOneThatPlays(t *testing.T) {
	_, err := rules.Seeded().Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}
