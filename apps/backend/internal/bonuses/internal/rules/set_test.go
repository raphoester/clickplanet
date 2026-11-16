package rules_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func rush() cprules.Snapshot[*bonusesv1.Rules] {
	return cprules.Snapshot[*bonusesv1.Rules]{Name: "rush", Rules: &bonusesv1.Rules{
		Shield: &bonusesv1.ShieldRules{MinPerBox: proto.Uint32(10), MaxPerBox: proto.Uint32(30)},
	}}
}

func restored(t *testing.T, snapshot cprules.Snapshot[*bonusesv1.Rules]) cprules.Set[*bonusesv1.Rules] {
	t.Helper()

	set, err := snapshot.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
	return set
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := cprules.Snapshot[*bonusesv1.Rules]{Name: "quiet", Rules: &bonusesv1.Rules{SignInRefill: proto.Bool(false)}}

	assert.False(t, restored(t, quiet).TakePlayedSnapshot().Rules.GetSignInRefill())
}

func TestADefaultThatCannotPlayIsRefusedWithTheValueNamed(t *testing.T) {
	broken := map[string]func(*bonusesv1.Rules){
		"kinds.bomb":                            func(r *bonusesv1.Rules) { r.Kinds.Bomb = proto.Float64(-1) },
		"kinds.refill is NaN":                   func(r *bonusesv1.Rules) { r.Kinds.Refill = proto.Float64(math.NaN()) },
		"kinds weigh every kind a box may be 0": func(r *bonusesv1.Rules) { r.Kinds = &bonusesv1.KindWeights{} },
		"spread.maxPerBox":                      func(r *bonusesv1.Rules) { r.Spread.MaxPerBox = proto.Uint32(0) },
		"enclose.maxPerBox":                     func(r *bonusesv1.Rules) { r.Enclose.MaxPerBox = proto.Uint32(0) },
		"shield.minPerBox":                      func(r *bonusesv1.Rules) { r.Shield.MinPerBox = proto.Uint32(0) },
		"shield.maxPerBox":                      func(r *bonusesv1.Rules) { r.Shield.MaxPerBox = proto.Uint32(4) },
		"gates.on with no kind gates.open opens weighed above 0": func(r *bonusesv1.Rules) {
			r.Gates.On = proto.Bool(true)
			r.Kinds = &bonusesv1.KindWeights{Bomb: proto.Float64(1), EncloseClicks: proto.Float64(1)}
		},
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
	narrower := cprules.Snapshot[*bonusesv1.Rules]{Name: "narrower", Rules: &bonusesv1.Rules{
		Shield: &bonusesv1.ShieldRules{MinPerBox: proto.Uint32(25)},
	}}

	_, err := narrower.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "shield.maxPerBox is 20, under shield.minPerBox (25)")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *bonusesv1.Rules) { r.Spread = nil }).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "spread.maxPerBox is 0")
}

func TestEveryReasonIsSaid(t *testing.T) {
	_, err := rules.Named(cprules.Default, func(r *bonusesv1.Rules) {
		r.Spread.MaxPerBox = proto.Uint32(0)
		r.Enclose = nil
	}).Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "spread.maxPerBox is 0")
	assert.Contains(t, err.Error(), "enclose.maxPerBox is 0")
}

func TestEveryValueThatCanBeZeroIsADefault(t *testing.T) {
	set := rules.Named(cprules.Default, func(r *bonusesv1.Rules) {
		r.Kinds.Bomb = proto.Float64(0)
		r.SignInRefill = nil
	})

	_, err := set.Restore(parts.All(), rules.Seeded())
	require.NoError(t, err)
}

func TestTheGatesAreOffUntilTheLevelsAreShown(t *testing.T) {
	played := bonuses.RulesOf(restored(t, rules.Seeded()).TakePlayedSnapshot().Rules)

	for _, kind := range bonuses.Kinds {
		assert.True(t, played.MayDraw(kind, nil), "%s", kind)
	}
}

func TestASetPlaysDefaultsGatesUnlessItHoldsItsOwn(t *testing.T) {
	battle := cprules.Snapshot[*bonusesv1.Rules]{Name: "battle", Rules: &bonusesv1.Rules{
		Gates: &bonusesv1.GateRules{On: proto.Bool(false)},
	}}
	gated := rules.Named(cprules.Default, func(r *bonusesv1.Rules) { r.Gates.On = proto.Bool(true) })

	set, err := battle.Restore(parts.All(), gated)
	require.NoError(t, err)
	assert.True(t, bonuses.RulesOf(set.TakePlayedSnapshot().Rules).MayDraw(bonuses.KindBomb, nil), "the battle keeps them off")

	plain, err := cprules.Snapshot[*bonusesv1.Rules]{Name: "plain", Rules: &bonusesv1.Rules{}}.Restore(parts.All(), gated)
	require.NoError(t, err)
	played := bonuses.RulesOf(plain.TakePlayedSnapshot().Rules)
	assert.False(t, played.MayDraw(bonuses.KindBomb, nil))
	assert.True(t, played.MayDraw(bonuses.KindRefill, nil), "default's open kinds")
}
