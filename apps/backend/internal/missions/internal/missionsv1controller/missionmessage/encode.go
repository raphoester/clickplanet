package missionmessage

import (
	"errors"
	"fmt"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
)

var ErrUnknownKind = errors.New("no message for this kind of mission")

func Mission(status missions.MissionStatus) (*missionsv1.Mission, error) {
	encoded := &missionsv1.Mission{
		Id:   status.Mission.ID.String(),
		Done: status.Done,
		Paid: status.Paid,
		Reward: &missionsv1.MissionReward{Kind: &missionsv1.MissionReward_Charges{Charges: &missionsv1.Charges{
			Kind: planetgrant.KindOf(status.Mission.Reward.Kind), Amount: status.Mission.Reward.Amount,
		}}},
	}

	switch order := status.Mission.Order.(type) {
	case missions.TakeFor:
		encoded.Kind = &missionsv1.Mission_TakeFor{TakeFor: &missionsv1.TakeFor{
			CountryId: order.Flag, Goal: order.Goal, Taken: status.Progress,
		}}
	case missions.TakeFrom:
		encoded.Kind = &missionsv1.Mission_TakeFrom{TakeFrom: &missionsv1.TakeFrom{
			CountryId: order.Flag, Goal: order.Goal, Taken: status.Progress,
		}}
	case missions.TakeIn:
		encoded.Kind = &missionsv1.Mission_TakeIn{TakeIn: &missionsv1.TakeIn{
			CountryId: order.Ground, Goal: order.Goal, Taken: status.Progress,
		}}
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownKind, order)
	}
	return encoded, nil
}
