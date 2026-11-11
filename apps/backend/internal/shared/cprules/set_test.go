package cprules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestASetIsWhatItWasMadeOf(t *testing.T) {
	kept := restored(t, rush()).TakeSnapshot()

	assert.Equal(t, cprules.SetName("rush"), kept.Name)
	assert.True(t, proto.Equal(rush().Rules, kept.Rules))
}

func TestASetKeepsNothingItsSnapshotsShare(t *testing.T) {
	kept := rush()
	played := restored(t, kept)

	kept.Rules.Clicks.Tempo = proto.Float64(9)
	played.TakeSnapshot().Rules.Clicks.Tempo = proto.Float64(99)
	played.TakePlayedSnapshot().Rules.Charges.Shield.PerTile = proto.Uint32(99)

	assert.True(t, played.Equal(restored(t, rush())))
}

func TestASetPlaysDefaultsValueWhereItHoldsNone(t *testing.T) {
	played := restored(t, rush()).TakePlayedSnapshot()

	assert.True(t, proto.Equal(named("rush", quicker).Rules, played.Rules), "%v", played.Rules)
}

func TestAZeroASetHoldsIsPlayed(t *testing.T) {
	quiet := snapshot{Name: "quiet", Rules: &planetv1.Rules{Frozen: proto.Bool(false)}}

	played, err := quiet.Restore(parts, named(cprules.Default, func(r *planetv1.Rules) { r.Frozen = proto.Bool(true) }))

	require.NoError(t, err)
	assert.False(t, played.TakePlayedSnapshot().Rules.GetFrozen())
	assert.Equal(t, frozen(false), frozenPart.Of(played))
}

func TestTwoSetsAreEqualWhenTheyPlayAlikeUnderOneName(t *testing.T) {
	assert.True(t, restored(t, rush()).Equal(restored(t, rush())))
	assert.True(t, restored(t, rush()).Equal(restored(t, named("rush", quicker))), "a set that holds less plays alike")
	assert.False(t, restored(t, rush()).Equal(restored(t, named("rush", asIs))))
	assert.False(t, restored(t, rush()).Equal(restored(t, snapshot{Name: "calm", Rules: rush().Rules})))
}

func TestASetIsCheckedOverDefault(t *testing.T) {
	_, err := snapshot{Name: "rush", Rules: &planetv1.Rules{Clicks: &planetv1.ClickRules{NewAccountClicks: proto.Float64(100)}}}.
		Restore(parts, seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.EqualError(t, err, "this rule set cannot play: clicks.newAccountClicks is 100, over burst (60)")
}

func TestDefaultPlaysOnlyWhatItHolds(t *testing.T) {
	_, err := named(cprules.Default, func(r *planetv1.Rules) { r.Clicks = nil }).Restore(parts, seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "clicks.tempo is 0")
}

func TestEveryReasonIsSaidNamedByItsPathInTheSet(t *testing.T) {
	_, err := named("rush", func(r *planetv1.Rules) {
		r.Clicks = &planetv1.ClickRules{Tempo: proto.Float64(0), Burst: proto.Uint32(0), NewAccountClicks: proto.Float64(0)}
		r.Charges.Shield.PerTile = proto.Uint32(0)
	}).Restore(parts, seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.EqualError(t, err, "this rule set cannot play: clicks.tempo is 0, and must be above 0; "+
		"clicks.burst is 0, and must be above 0; charges.shield.perTile is 0, and must be above 0",
		"a root part's reasons take no key before them")
}

func TestOnlyDefaultIsRefusedWhatDefaultMayNotPlay(t *testing.T) {
	on := func(r *planetv1.Rules) { r.Arena.On = proto.Bool(true) }

	_, err := named(cprules.Default, on).Restore(parts, seeded())
	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "arena.on: default never plays in an arena")

	_, err = named("battle", on).Restore(parts, seeded())
	require.NoError(t, err)
}

func TestANameThatIsNotALowercaseWordIsRefused(t *testing.T) {
	for _, name := range []cprules.SetName{"", "Rush", "a234567890123456789012345678901234"} {
		_, err := named(name, asIs).Restore(parts, seeded())
		require.ErrorIs(t, err, cprules.ErrInvalidSet, name)
		assert.Contains(t, err.Error(), "a name is a lowercase letter", name)
	}
}

func TestASetThatHoldsNothingPlaysDefault(t *testing.T) {
	played := restored(t, snapshot{Name: "calm"}).TakePlayedSnapshot()

	assert.True(t, proto.Equal(seeded().Rules, played.Rules), "%v", played.Rules)
}
