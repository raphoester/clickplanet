package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
)

func TestDefaultNeverPlaysASiege(t *testing.T) {
	sieging := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Siege.On = new(true) })

	_, err := sieging.Restore(parts.All(), sieging)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.ErrorContains(t, err, "siege.on: default never plays a siege")
}

func TestASetPlaysASiegeOverDefault(t *testing.T) {
	siege := rules.Snapshot{Name: "siege", Rules: &planetv1.Rules{Siege: &planetv1.SiegeRules{On: new(true)}}}

	set, err := siege.Restore(parts.All(), rules.Seeded())

	require.NoError(t, err)
	played := set.TakePlayedSnapshot().Rules.GetSiege()
	assert.True(t, played.GetOn())
	assert.Equal(t, 8*time.Second, played.GetFreeze().AsDuration(), "default's freeze")
}

func TestAFreezePast30SecondsIsRefused(t *testing.T) {
	long := rules.Snapshot{Name: "siege", Rules: &planetv1.Rules{Siege: &planetv1.SiegeRules{Freeze: durationpb.New(31 * time.Second)}}}

	_, err := long.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.ErrorContains(t, err, "siege.freeze is 31s, outside 0s to 30s")
}
