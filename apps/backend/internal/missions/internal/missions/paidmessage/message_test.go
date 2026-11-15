package paidmessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/paidmessage"
)

var ada = missions.AccountID{15: 1}

func TestTheTotalsAreToldWithTheirKindsInTheOrderOfMissionKind(t *testing.T) {
	told := paidmessage.Of(ada, missions.Paid{InAll: 7, ByKind: map[missions.Kind]missions.KindPaid{
		missions.KindFortifyTiles: {Done: 1, Goals: 40},
		missions.KindTakeFor:      {Done: 4, Goals: 640},
		missions.KindCatchBoxes:   {Done: 2, Goals: 7},
	}})

	assert.True(t, proto.Equal(&missionsv1.MissionsPaid{AccountId: ada.String(), PaidInAll: 7, ByKind: []*missionsv1.KindPaid{
		{Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, Done: 4, Goals: 640},
		{Kind: missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES, Done: 2, Goals: 7},
		{Kind: missionsv1.MissionKind_MISSION_KIND_FORTIFY_TILES, Done: 1, Goals: 40},
	}}, told), told)
}

func TestNothingPaidIsToldAsZeroWithNoKind(t *testing.T) {
	assert.True(t, proto.Equal(&missionsv1.MissionsPaid{AccountId: ada.String()}, paidmessage.Of(ada, missions.NothingPaid())))
}

func TestEveryKindIsAMissionKindOfItsOwnAndEveryMissionKindIsOne(t *testing.T) {
	told := map[missionsv1.MissionKind]missions.Kind{}
	for _, kind := range missions.TestingKinds() {
		encoded := paidmessage.KindOf(kind)
		assert.NotEqual(t, missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED, encoded, kind)
		assert.NotContains(t, told, encoded, "%s and %s", kind, told[encoded])
		told[encoded] = kind
	}

	values := missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED.Descriptor().Values()
	for i := range values.Len() {
		kind := missionsv1.MissionKind(values.Get(i).Number())
		if kind != missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED {
			assert.Contains(t, told, kind, "no kind of mission is told as %s", kind)
		}
	}
}
