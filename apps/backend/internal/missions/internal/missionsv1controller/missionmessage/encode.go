package missionmessage

import (
	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

func Mission(mission missions.MissionSnapshot, sheet missions.SheetSnapshot) *missionsv1.Mission {
	return &missionsv1.Mission{
		Id:              mission.ID.String(),
		Kind:            MissionKind(mission.Kind),
		TargetCountryId: mission.Target,
		Goal:            mission.Goal,
		Progress:        sheet.Taken[mission.ID],
		Collected:       sheet.Collected.Contains(mission.ID),
		RewardKind:      RewardKind(mission.Reward.Kind),
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

func RewardKind(kind missions.RewardKind) planetv1.BonusKind {
	switch kind {
	case missions.RewardShields:
		return planetv1.BonusKind_BONUS_KIND_SHIELDS
	case missions.RewardSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case missions.RewardBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	}
	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
