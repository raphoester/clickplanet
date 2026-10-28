package scoringgrant_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/scoringgrant"
)

func TestAMissionDoneGivesItsFlagItsPointsAtTheTimeItWasDone(t *testing.T) {
	at := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	mission := flagmissions.MissionID{15: 7}

	grants := scoringgrant.Of("fr", []flagmissions.Done{{Mission: mission, Points: 2}}, at)

	require.Len(t, grants, 1)
	assert.True(t, proto.Equal(&scoringv1.GrantPoints{
		GrantId: mission.String(), Country: "fr", Points: 2, GrantedAt: timestamppb.New(at),
	}, grants[0]))
}
