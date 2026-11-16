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

func TestARallyIsToldWithTheGroundAndTheLandmassOfItsTile(t *testing.T) {
	calledAt := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	told := facts.NewLog(cptime.NewFixedClock(calledAt))
	rally := rallies.Snapshot{
		ID: rallies.RallyID(uuid.MustParse("00000000-0000-7000-8000-000000000001")), Flag: "fr", Tile: 8, Radius: 0.072,
		Account:  cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")),
		CalledAt: calledAt, EndsAt: calledAt.Add(3 * time.Minute),
	}.Restore()

	rally_facts.New(told, tasmania{}).Called(rally)

	taken := told.Take()
	require.Len(t, taken, 1)
	assert.True(t, proto.Equal(&planetv1.RallyCalled{
		RallyId: "00000000-0000-7000-8000-000000000001", AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", Country: "fr",
		TileId: 8, Ground: "au", LandmassId: 420, CalledAt: timestamppb.New(calledAt),
	}, taken[0]), "%v", taken[0])
}
