package rally_facts_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies/rally_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type tasmania struct{}

func (tasmania) CountryOf(uint32) string             { return "au" }
func (tasmania) LandmassOf(uint32) clicks.LandmassID { return 420 }

type nearest struct {
	tile uint32
	arc  float64
}

func (n nearest) Nearest(clicks.Vec3) (uint32, float64) { return n.tile, n.arc }

var calledAt = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func called(t *testing.T, near nearest) *planetv1.RallyCalled {
	t.Helper()
	told := facts.NewLog(cptime.NewFixedClock(calledAt))
	rally := rallies.Snapshot{
		ID: rallies.RallyID(uuid.MustParse("00000000-0000-7000-8000-000000000001")), Flag: "fr",
		Centre: clicks.Vec3{X: 0.6, Z: 0.8}, Radius: 0.072,
		Account:  cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")),
		CalledAt: calledAt, EndsAt: calledAt.Add(3 * time.Minute),
	}.Restore()

	rally_facts.New(told, tasmania{}, near).Called(rally)

	taken := told.Take()
	require.Len(t, taken, 1)
	message, ok := taken[0].(*planetv1.RallyCalled)
	require.True(t, ok, "%T", taken[0])
	return message
}

func TestARallyIsToldWithTheGroundAndTheLandmassOfTheTileNearestItsCentre(t *testing.T) {
	message := called(t, nearest{tile: 8, arc: 0.07})

	assert.True(t, proto.Equal(&planetv1.RallyCalled{
		RallyId: "00000000-0000-7000-8000-000000000001", AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", Country: "fr",
		Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}, Ground: "au", LandmassId: 420, CalledAt: timestamppb.New(calledAt),
	}, message), "%v", message)
}

func TestARallyWithNoTileInsideItsRingNamesNoGround(t *testing.T) {
	message := called(t, nearest{tile: 8, arc: 0.08})

	assert.Empty(t, message.GetGround())
	assert.Zero(t, message.GetLandmassId())
	assert.True(t, proto.Equal(&planetv1.GlobePoint{X: 0.6, Z: 0.8}, message.GetPoint()))
}
