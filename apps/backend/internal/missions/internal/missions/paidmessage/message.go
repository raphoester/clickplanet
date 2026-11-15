package paidmessage

import (
	"cmp"
	"slices"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

// What an account was paid in all as the missions.v1.MissionsPaid event, its kinds in the order of MissionKind.
func Of(account missions.AccountID, paid missions.Paid) *missionsv1.MissionsPaid {
	told := &missionsv1.MissionsPaid{AccountId: account.String(), PaidInAll: paid.InAll}
	for kind, total := range paid.ByKind {
		told.ByKind = append(told.ByKind, &missionsv1.KindPaid{Kind: KindOf(kind), Done: total.Done, Goals: total.Goals})
	}
	slices.SortFunc(told.ByKind, func(a, b *missionsv1.KindPaid) int { return cmp.Compare(a.GetKind(), b.GetKind()) })
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
