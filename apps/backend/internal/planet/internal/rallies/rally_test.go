package rallies_test

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	calledAt = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	alice    = cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"))
	bob      = cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"))
)

// rallyOf is a rally of 3 minutes from calledAt around the Line ground's tile, its ring 5 hundredths of a radian wide.
func rallyOf(account cpsession.AccountID, flag string, tile uint32, at time.Time) rallies.Rally {
	centre, _ := rallies.Line{}.Position(tile)
	return rallies.Snapshot{
		ID: rallies.RallyID(uuid.MustParse("00000000-0000-7000-8000-000000000001")), Account: account, Flag: flag,
		Centre: centre, Radius: 0.05, CalledAt: at, EndsAt: at.Add(3 * time.Minute),
	}.Restore()
}

func TestARingCoversEveryTileWithinItsRadiusOfItsCentre(t *testing.T) {
	rally := rallyOf(alice, "fr", 10, calledAt)

	for _, tile := range []uint32{5, 8, 10, 12, 15} {
		assert.True(t, rally.Covers(tile, rallies.Line{}), "tile %d", tile)
	}
	for _, tile := range []uint32{4, 16, 80} {
		assert.False(t, rally.Covers(tile, rallies.Line{}), "tile %d", tile)
	}
	assert.False(t, rally.Covers(0, rallies.Line{}), "a tile off the map")
}

func TestARingIsAroundAPointOfTheGlobeNotATile(t *testing.T) {
	between := rallies.Snapshot{Centre: clicks.Vec3{X: math.Cos(0.095), Y: math.Sin(0.095)}, Radius: 0.05, EndsAt: calledAt}.Restore()

	assert.True(t, between.Covers(6, rallies.Line{}))
	assert.True(t, between.Covers(15, rallies.Line{}))
	assert.False(t, between.Covers(5, rallies.Line{}))
}

func TestAPointIsAnyVectorOfSomeLength(t *testing.T) {
	point, err := rallies.PointOf(clicks.Vec3{Z: 3})
	require.NoError(t, err)
	assert.Equal(t, clicks.Vec3{Z: 1}, point, "on the unit globe")

	for _, none := range []clicks.Vec3{{}, {X: math.NaN()}, {Y: math.Inf(1)}} {
		_, err := rallies.PointOf(none)
		require.ErrorIs(t, err, rallies.ErrNoPoint)
	}
}
