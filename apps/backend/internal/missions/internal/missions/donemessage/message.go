package donemessage

import (
	"cmp"
	"slices"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

// What an account has done in all as the missions.v1.MissionsDone event, its kinds in the order of MissionKind.
func Of(account missions.AccountID, done missions.Done) *missionsv1.MissionsDone {
	told := &missionsv1.MissionsDone{AccountId: account.String(), DoneInAll: done.InAll}
	for kind, total := range done.ByKind {
		told.ByKind = append(told.ByKind, &missionsv1.KindDone{Kind: KindOf(kind), Done: total.Done, Goals: total.Goals})
	}
	slices.SortFunc(told.ByKind, func(a, b *missionsv1.KindDone) int { return cmp.Compare(a.GetKind(), b.GetKind()) })
	return told
}

func KindOf(kind missions.Kind) missionsv1.MissionKind {
	switch kind {
	case missions.KindTakeFor:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_FOR
	case missions.KindTakeFrom:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_FROM
	case missions.KindTakeIn:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_IN
	case missions.KindTakeFromIn:
		return missionsv1.MissionKind_MISSION_KIND_TAKE_FROM_IN
	case missions.KindCatchBoxes:
		return missionsv1.MissionKind_MISSION_KIND_CATCH_BOXES
	case missions.KindAnswerQuizzes:
		return missionsv1.MissionKind_MISSION_KIND_ANSWER_QUIZZES
	case missions.KindBombIn:
		return missionsv1.MissionKind_MISSION_KIND_BOMB_IN
	case missions.KindShieldTiles:
		return missionsv1.MissionKind_MISSION_KIND_SHIELD_TILES
	case missions.KindFortifyTiles:
		return missionsv1.MissionKind_MISSION_KIND_FORTIFY_TILES
	}
	return missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED
}
