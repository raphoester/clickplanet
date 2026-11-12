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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func rush() cprules.Snapshot[*planetv1.Rules] {
	return cprules.Snapshot[*planetv1.Rules]{Name: "rush", Rules: &planetv1.Rules{
		Clicks:  &planetv1.ClickRules{Tempo: proto.Float64(3)},
		Charges: &planetv1.ChargeRules{Spread: &planetv1.SpreadRules{Clicks: proto.Uint32(30)}},
	}}
}

func restored(t *testing.T, snapshot cprules.Snapshot[*planetv1.Rules]) cprules.Set[*planetv1.Rules] {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := cprules.Snapshot[*planetv1.Rules]{Name: "quiet", Rules: &planetv1.Rules{
		Charges: &planetv1.ChargeRules{Bomb: &planetv1.BombRules{Off: proto.Bool(false)}},
		Toll:    &planetv1.TollRules{},
	}}

	played := restored(t, quiet).TakePlayedSnapshot().Rules
	assert.False(t, played.GetCharges().GetBomb().GetOff())
	assert.Empty(t, played.GetToll().GetSteps())
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
		"charges.spread.rings is 0":     func(r *planetv1.Rules) { r.Charges.Spread.Rings = proto.Uint32(0) },
		"charges.spread.rings is 4":     func(r *planetv1.Rules) { r.Charges.Spread.Rings = proto.Uint32(4) },
		"charges.bomb.rings":            func(r *planetv1.Rules) { r.Charges.Bomb = nil },
		"charges.enclose.maxTiles":      func(r *planetv1.Rules) { r.Charges.Enclose.MaxTiles = proto.Uint32(0) },
		"charges.enclose.held":          func(r *planetv1.Rules) { r.Charges.Enclose.Held = proto.Uint32(0) },
		"charges.shield.held":           func(r *planetv1.Rules) { r.Charges.Shield.Held = proto.Uint32(0) },
		"charges.shield.perTile is 0":   func(r *planetv1.Rules) { r.Charges.Shield.PerTile = proto.Uint32(0) },
		"charges.shield.perTile is 256": func(r *planetv1.Rules) { r.Charges.Shield.PerTile = proto.Uint32(256) },
	}
	for reason, breaks := range broken {
		t.Run(reason, func(t *testing.T) {
			_, err := rules.Named(cprules.Default, breaks).Restore(parts.All(), rules.Seeded())

			require.ErrorIs(t, err, cprules.ErrInvalidSet)
			assert.Contains(t, err.Error(), reason)
		})
	}
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(cprules.Default, func(r *planetv1.Rules) {
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
