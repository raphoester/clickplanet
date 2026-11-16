package missions

import "maps"

// A kind of mission, one per Order type: what the missions done are counted by.
type Kind string

const (
	KindTakeFor       Kind = "take_for"
	KindTakeFrom      Kind = "take_from"
	KindTakeIn        Kind = "take_in"
	KindTakeFromIn    Kind = "take_from_in"
	KindCatchBoxes    Kind = "catch_boxes"
	KindAnswerQuizzes Kind = "answer_quizzes"
	KindBombIn        Kind = "bomb_in"
	KindShieldTiles   Kind = "shield_tiles"
	KindFortifyTiles  Kind = "fortify_tiles"
)

// What an account has done in all, every day: how many missions, and by kind how many and their goals summed, whether
// or not they were paid. Totals, not one fact per mission: told twice or late, they change nothing, and the first one
// told brings every mission done before.
type Done struct {
	InAll  uint64
	ByKind map[Kind]KindDone
}

// The missions of one kind an account has done in all.
type KindDone struct {
	Done  uint64
	Goals uint64
}

func NothingDone() Done {
	return Done{ByKind: map[Kind]KindDone{}}
}

// A day's missions done added: each mission of the brief whose progress on the sheet reached its goal.
func (d Done) WithDay(brief Brief, sheet Sheet) Done {
	next := Done{InAll: d.InAll, ByKind: maps.Clone(d.ByKind)}
	if next.ByKind == nil {
		next.ByKind = map[Kind]KindDone{}
	}
	for _, mission := range brief.missions {
		if !mission.order.done(sheet.progress[mission.id]) {
			continue
		}
		kind := mission.order.kind()
		next.InAll++
		next.ByKind[kind] = KindDone{
			Done: next.ByKind[kind].Done + 1, Goals: next.ByKind[kind].Goals + uint64(mission.order.goal()),
		}
	}
	return next
}

// The account beside what it has done, for a read of many accounts.
type AccountDone struct {
	Account AccountID
	Done    Done
}
