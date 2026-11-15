package missions

import "maps"

// A kind of mission, one per Order type: what the missions paid are counted by.
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

// What an account was paid in all, every day: how many missions, and by kind how many and their goals summed. Totals,
// not one fact per payment: told twice or late, they change nothing, and the first one told brings every payment before.
type Paid struct {
	InAll  uint64
	ByKind map[Kind]KindPaid
}

// The missions of one kind an account was paid in all.
type KindPaid struct {
	Done  uint64
	Goals uint64
}

func NothingPaid() Paid {
	return Paid{ByKind: map[Kind]KindPaid{}}
}

// A day's payments added: every mission the sheet paid, and by kind each one the brief holds.
func (p Paid) WithDay(brief Brief, sheet Sheet) Paid {
	next := Paid{InAll: p.InAll + uint64(sheet.paid.Len()), ByKind: maps.Clone(p.ByKind)} //nolint:gosec // a length is never negative.
	if next.ByKind == nil {
		next.ByKind = map[Kind]KindPaid{}
	}
	for _, mission := range brief.missions {
		if !sheet.paid.Contains(mission.id) {
			continue
		}
		kind := mission.order.kind()
		next.ByKind[kind] = KindPaid{
			Done: next.ByKind[kind].Done + 1, Goals: next.ByKind[kind].Goals + uint64(mission.order.goal()),
		}
	}
	return next
}

// The account it was paid to beside what it was paid, for a read of many accounts.
type AccountPaid struct {
	Account AccountID
	Paid    Paid
}
