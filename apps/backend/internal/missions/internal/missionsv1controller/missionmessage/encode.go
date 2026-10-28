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
	case missions.TakeFromIn:
		encoded.Kind = &missionsv1.Mission_TakeFromIn{TakeFromIn: &missionsv1.TakeFromIn{
			CountryId: order.Flag, GroundId: order.Ground, Goal: order.Goal, Taken: status.Progress,
		}}
	case missions.CatchBoxes:
		encoded.Kind = &missionsv1.Mission_CatchBoxes{CatchBoxes: &missionsv1.CatchBoxes{
			Goal: order.Goal, Caught: status.Progress,
		}}
	case missions.AnswerQuizzes:
		encoded.Kind = &missionsv1.Mission_AnswerQuizzes{AnswerQuizzes: &missionsv1.AnswerQuizzes{
			Goal: order.Goal, Answered: status.Progress,
		}}
	case missions.BombIn:
		encoded.Kind = &missionsv1.Mission_BombIn{BombIn: &missionsv1.BombIn{
			CountryId: order.Ground, Goal: order.Goal, Dropped: status.Progress,
		}}
	case missions.ShieldTiles:
		encoded.Kind = &missionsv1.Mission_ShieldTiles{ShieldTiles: &missionsv1.ShieldTiles{
			Shields: order.Shields, Goal: order.Goal, Shielded: status.Progress,
		}}
	case missions.FortifyTiles:
		encoded.Kind = &missionsv1.Mission_FortifyTiles{FortifyTiles: &missionsv1.FortifyTiles{
			Goal: order.Goal, Fortified: status.Progress,
		}}
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnknownKind, order)
	}
	return encoded, nil
}
