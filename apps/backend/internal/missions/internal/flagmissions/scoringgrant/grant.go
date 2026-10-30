package scoringgrant

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

// A mission is done once, so its id names the grant, which scoring grants once.
func Of(flag string, done []flagmissions.Done, at time.Time) []proto.Message {
	grants := make([]proto.Message, 0, len(done))
	for _, mission := range done {
		grants = append(grants, &scoringv1.GrantPoints{
			GrantId: mission.Mission.String(), Country: flag, Points: mission.Points, GrantedAt: timestamppb.New(at),
		})
	}
	return grants
}
