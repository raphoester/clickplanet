package arenas_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// tagged is a seed that says which set it was named by.
type tagged struct{ set string }

func (tagged) Tiles() []clicks.StartTile { return nil }

type namedSeeds struct{}

func (namedSeeds) SeedOf(set cprules.Set[*planetv1.Rules]) arenas.Seed {
	return tagged{set: string(set.TakeSnapshot().Name)}
}

type failingSets struct{}

func (failingSets) Set(context.Context, cprules.SetName) (cprules.Set[*planetv1.Rules], error) {
	return cprules.Set[*planetv1.Rules]{}, errors.New("postgres is restarting")
}

func TestAFailedReadOfTheSetIsNoRefusal(t *testing.T) {
	_, err := arenas.NewSetups(failingSets{}, namedSeeds{}).Setup(t.Context(), "siege")

	require.Error(t, err)
	assert.NotErrorIs(t, err, arenas.ErrRefused, "asked again, it may apply")
}
