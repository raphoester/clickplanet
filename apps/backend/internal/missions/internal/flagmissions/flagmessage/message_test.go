package flagmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/flagmessage"
)

var (
	at      = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	mission = flagmissions.MissionID{15: 7}
)

type unknownOrder struct{ flagmissions.TakeIn }

func TestAMissionDoneIsToldWithItsFlagAsItsPlayersSeeItAtTheTimeItWasDone(t *testing.T) {
	told, err := flagmessage.Done("fr", []flagmissions.Done{{
		Mission:  flagmissions.MissionSnapshot{ID: mission, Order: flagmissions.BreakShields{Flag: "de", Goal: 300}, Points: 6},
		Progress: 300,
		At:       at,
	}})

	require.NoError(t, err)
	require.Len(t, told, 1)
	assert.True(t, proto.Equal(&missionsv1.FlagMissionDone{
		Flag: "fr",
		Mission: &missionsv1.FlagMission{Id: mission.String(), Points: 6, Done: true, Kind: &missionsv1.FlagMission_BreakShields{
			BreakShields: &missionsv1.BreakShields{CountryId: "de", Goal: 300, Broken: 300},
		}},
		DoneAt: timestamppb.New(at),
	}, told[0]))
}

func TestAKindWithNoMessageIsAnError(t *testing.T) {
	_, err := flagmessage.Done("fr", []flagmissions.Done{{
		Mission: flagmissions.MissionSnapshot{ID: mission, Order: unknownOrder{}, Points: 6}, At: at,
	}})

	assert.ErrorIs(t, err, flagmessage.ErrUnknownKind)
}
