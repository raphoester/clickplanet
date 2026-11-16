package planetwords_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/planetwords"
)

func TestEveryKindOnTheLadderIsOneOfPlanets(t *testing.T) {
	for _, unlock := range (levels.Ladder{}).Unlocks() {
		_, err := planetwords.BonusKind(unlock.Kind())
		require.NoError(t, err, "%s", unlock.Kind())
	}
}

func TestAKindPlanetDoesNotNameIsRefused(t *testing.T) {
	for _, kind := range []levels.Kind{"", "unspecified", "rally"} {
		_, err := planetwords.BonusKind(kind)
		require.ErrorIs(t, err, planetwords.ErrUnknownKind, "%q", kind)
	}
}

func TestAKindIsPlanetsWordInUpperCase(t *testing.T) {
	kind, err := planetwords.BonusKind("enclose_clicks")
	require.NoError(t, err)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, kind)
}
