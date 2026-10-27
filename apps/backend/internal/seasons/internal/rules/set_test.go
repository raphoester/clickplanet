package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
)

func restored(t *testing.T, snapshot rules.Snapshot) rules.Set {
	t.Helper()

	set, err := snapshot.Restore()
	require.NoError(t, err)
	return set
}

func TestASetIsWhatItWasMadeOf(t *testing.T) {
	set := restored(t, rules.Battle("rush", 3))

	kept := set.TakeSnapshot()
	assert.Equal(t, rules.SetName("rush"), kept.Name)
	assert.True(t, proto.Equal(rules.Battle("rush", 3).Rules, kept.Rules))
	assert.Equal(t, rules.OwnRound{Enabled: true, Times: 3}, set.OwnRound())
}

func TestDefaultHasNoRoundOfItsOwn(t *testing.T) {
	assert.Equal(t, rules.OwnRound{Enabled: false, Times: 1}, restored(t, rules.Seeded()).OwnRound())
}

func TestASetKeepsNothingItsSnapshotsShare(t *testing.T) {
	snapshot := rules.Battle("rush", 3)
	set := restored(t, snapshot)

	snapshot.Rules.OwnRound.Times = 9
	set.TakeSnapshot().Rules.OwnRound.Enabled = false

	assert.True(t, set.Equal(restored(t, rules.Battle("rush", 3))))
}

func TestTwoSetsAreEqualWhenTheyCountAlikeUnderOneName(t *testing.T) {
	assert.True(t, restored(t, rules.Battle("rush", 3)).Equal(restored(t, rules.Battle("rush", 3))))
	assert.False(t, restored(t, rules.Battle("rush", 3)).Equal(restored(t, rules.Battle("rush", 2))))
	assert.False(t, restored(t, rules.Battle("rush", 3)).Equal(restored(t, rules.Battle("calm", 3))))
}

func TestPointsTimesOutsideOneToTenAreRefused(t *testing.T) {
	for _, times := range []uint32{0, 11} {
		_, err := rules.Battle("rush", times).Restore()

		require.ErrorIs(t, err, rules.ErrInvalidSet)
		assert.Contains(t, err.Error(), "ownRound.times")
	}
}

func TestARoundOfItsOwnLeftOutIsRefused(t *testing.T) {
	_, err := rules.Snapshot{Name: "rush", Rules: &seasonsv1.Rules{}}.Restore()

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "ownRound.times is 0")
}

func TestANameThatIsNoSetsIsRefused(t *testing.T) {
	_, err := rules.Battle("Rush!", 3).Restore()

	require.ErrorIs(t, err, rules.ErrInvalidSet)
}

func TestOnlyALaterDecisionOverrulesTheChoiceKept(t *testing.T) {
	at := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	kept := rules.Choice{Set: "rush", DecidedAt: at}

	assert.True(t, rules.Choice{Set: rules.Default, DecidedAt: at.Add(time.Second)}.Overrules(kept))
	assert.False(t, rules.Choice{Set: rules.Default, DecidedAt: at}.Overrules(kept))
	assert.False(t, rules.Choice{Set: rules.Default, DecidedAt: at.Add(-time.Second)}.Overrules(kept))
}
