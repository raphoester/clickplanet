package arenas_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
)

// tagged is a seed that says which set it was named by.
type tagged struct{ set string }

func (tagged) Tiles() []clicks.StartTile { return nil }

type namedSeeds struct{}

func (namedSeeds) SeedOf(set rules.Set) arenas.Seed {
	return tagged{set: string(set.TakeSnapshot().Name)}
}

type failingSets struct{}

func (failingSets) Set(context.Context, rules.SetName) (rules.Set, error) {
	return rules.Set{}, errors.New("postgres is restarting")
}

func TestAnArenaIsSetUpFromTheSetItIsOpenedFor(t *testing.T) {
	sets := rules.NewMemoryStore(parts.All(), rules.Seeded(), rules.Named("siege", func(r *planetv1.Rules) {
		r.Arena = &planetv1.ArenaRules{On: new(true), Freeze: durationpb.New(5 * time.Second)}
	}))

	setup, err := arenas.NewSetups(sets, namedSeeds{}).Setup(t.Context(), "siege")

	require.NoError(t, err)
	assert.Equal(t, arenas.Setup{Freeze: 5 * time.Second, Seed: tagged{set: "siege"}}, setup)
}

func TestAnArenaOfASetThatCannotPlayIsRefused(t *testing.T) {
	sets := rules.NewMemoryStore(parts.All(), rules.Seeded())

	_, err := arenas.NewSetups(sets, namedSeeds{}).Setup(t.Context(), "gone")

	require.ErrorIs(t, err, arenas.ErrRefused)
}

func TestAFailedReadOfTheSetIsNoRefusal(t *testing.T) {
	_, err := arenas.NewSetups(failingSets{}, namedSeeds{}).Setup(t.Context(), "siege")

	require.Error(t, err)
	assert.NotErrorIs(t, err, arenas.ErrRefused, "asked again, it may apply")
}
