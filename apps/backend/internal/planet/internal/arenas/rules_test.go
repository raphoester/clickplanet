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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestDefaultNeverPlaysAnArena(t *testing.T) {
	sieging := rules.Named(cprules.Default, func(r *planetv1.Rules) { r.Arena.On = new(true) })

	_, err := sieging.Restore(parts.All(), sieging)

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.ErrorContains(t, err, "arena.on: default never plays in an arena")
}

func TestASetPlaysAnArenaOverDefault(t *testing.T) {
	arena := cprules.Snapshot[*planetv1.Rules]{Name: "battle", Rules: &planetv1.Rules{Arena: &planetv1.ArenaRules{On: new(true)}}}

	set, err := arena.Restore(parts.All(), rules.Seeded())

	require.NoError(t, err)
	played := set.TakePlayedSnapshot().Rules.GetArena()
	assert.True(t, played.GetOn())
	assert.Equal(t, 8*time.Second, played.GetFreeze().AsDuration(), "default's freeze")
}

func TestAFreezePast30SecondsIsRefused(t *testing.T) {
	long := cprules.Snapshot[*planetv1.Rules]{Name: "battle", Rules: &planetv1.Rules{
		Arena: &planetv1.ArenaRules{Freeze: durationpb.New(31 * time.Second)},
	}}

	_, err := long.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.ErrorContains(t, err, "arena.freeze is 31s, outside 0s to 30s")
}
