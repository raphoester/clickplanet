package missionmessage

import (
	"errors"
	"fmt"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
)

var ErrUnknownKind = errors.New("no message for this kind of mission")

func Day(day get_missions_usecase.Out) (*missionsv1.GetMissionsResponse, error) {
	encoded := &missionsv1.GetMissionsResponse{}
	if ends := day.Day.Snapshot().EndsAt; !ends.IsZero() {
		encoded.EndsAtUnixMs = ends.UnixMilli()
	}
	for _, status := range day.Sheet.Statuses(day.Brief) {
		mission, err := Mission(status)
		if err != nil {
			return nil, err
		}
		encoded.Missions = append(encoded.Missions, mission)
	}
	if status, ok := day.Flag.Status(); ok {
		flag, err := flagmessage.Mission(status)
		if err != nil {
			return nil, fmt.Errorf("failed to encode the flag's mission: %w", err)
		}
		encoded.FlagMission = flag
	}
	return encoded, nil
}

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
