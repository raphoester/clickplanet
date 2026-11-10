package seeds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
)

var ground = seeds.Ground{Capitals: clicks.NewCapitals(map[string]uint32{"fr": 50, "de": 60})}

func capitalsSeed(shields *uint32) *planetv1.Seed {
	return &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: shields}}}
}

func played(t *testing.T, kept rules.Snapshot, defaults rules.Snapshot) rules.Set {
	t.Helper()
	set, err := kept.Restore(parts.All(), defaults)
	require.NoError(t, err)
	return set
}

func TestDefaultSeedsEachCapitalWithItsShields(t *testing.T) {
	set := played(t, rules.Snapshot{Name: "battle", Rules: &planetv1.Rules{}}, rules.Seeded())

	seed := seeds.NewCatalog(ground).SeedOf(set)

	assert.Equal(t, []clicks.StartTile{{Tile: 60, Country: "de", Shields: 100}, {Tile: 50, Country: "fr", Shields: 100}},
		seed.Tiles())
}

func TestASetsOwnShieldsSeedTheCapitals(t *testing.T) {
	set := played(t, rules.Snapshot{Name: "battle", Rules: &planetv1.Rules{Seed: capitalsSeed(new(uint32(40)))}},
		rules.Seeded())

	assert.Equal(t, seeds.Capitals{Shields: 40}, seeds.Part.Of(set))
}

func TestASetHoldingThePatternAloneKeepsDefaultsNumbers(t *testing.T) {
	set := played(t, rules.Snapshot{Name: "battle", Rules: &planetv1.Rules{Seed: capitalsSeed(nil)}}, rules.Seeded())

	assert.Equal(t, seeds.Capitals{Shields: 100}, seeds.Part.Of(set))
}

func TestMoreCapitalShieldsThanATileKeepsAreRefused(t *testing.T) {
	many := rules.Snapshot{Name: "battle", Rules: &planetv1.Rules{Seed: capitalsSeed(new(uint32(256)))}}

	_, err := many.Restore(parts.All(), rules.Seeded())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.ErrorContains(t, err, "seed.capitals.shields is 256, over what a tile keeps (255)")
}

func TestADefaultThatNamesNoPatternCannotPlay(t *testing.T) {
	bare := rules.Named(rules.Default, func(r *planetv1.Rules) { r.Seed = nil })

	_, err := bare.Restore(parts.All(), bare)

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.ErrorContains(t, err, "seed.pattern: none is named")
}
