package missionmessage

import (
	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
)

func Mission(mission missions.MissionSnapshot, sheet missions.SheetSnapshot) *missionsv1.Mission {
	return &missionsv1.Mission{
		Id:              mission.ID.String(),
		Kind:            MissionKind(mission.Kind),
		TargetCountryId: mission.Target,
		Goal:            mission.Goal,
		Progress:        sheet.Taken[mission.ID],
		Paid:            sheet.Paid.Contains(mission.ID),
		RewardKind:      planetgrant.KindOf(mission.Reward.Kind),
		RewardAmount:    mission.Reward.Amount,
	}
}

func MissionKind(kind missions.Kind) missionsv1.MissionKind {
	switch kind {
	case missions.KindTakeFor:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_FOR
	case missions.KindTakeFrom:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_FROM
	case missions.KindTakeIn:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_IN
	}
	return missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED
}
