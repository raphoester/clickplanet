package missions

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type SheetSnapshot struct {
	Flag       string
	Taken      map[MissionID]uint32
	Paid       *cpcolls.Set[MissionID]
	TakesBelow uint64
}

type Sheet struct {
	flag       string
	taken      map[MissionID]uint32
	paid       *cpcolls.Set[MissionID]
	takesBelow uint64
}

func SheetOf(snapshot SheetSnapshot) Sheet {
	sheet := Sheet{flag: snapshot.Flag, taken: map[MissionID]uint32{}, paid: cpcolls.NewSet[MissionID](), takesBelow: snapshot.TakesBelow}
	for id, taken := range snapshot.Taken {
		sheet.taken[id] = taken
	}
	if snapshot.Paid != nil {
		sheet.paid.AddSet(snapshot.Paid)
	}
	return sheet
}

func (s Sheet) clone() Sheet {
	return SheetOf(SheetSnapshot{Flag: s.flag, Taken: s.taken, Paid: s.paid, TakesBelow: s.takesBelow})
}

func (s Sheet) Snapshot() SheetSnapshot {
	copied := s.clone()
	return SheetSnapshot{Flag: copied.flag, Taken: copied.taken, Paid: copied.paid, TakesBelow: copied.takesBelow}
}

func (s Sheet) Enlisted() bool { return s.flag != "" }

// A take can be told twice: one below the mark was counted already. Takes come in ledger order.
func (s Sheet) WithTake(take Take, brief Brief) Sheet {
	if take.Position < s.takesBelow || s.flag != "" && take.Flag != s.flag {
		return s
	}

	next := s.clone()
	next.flag = take.Flag
	changed := s.flag == ""
	for _, mission := range brief.missions {
		if mission.Counts(take) && next.taken[mission.id] < mission.goal {
			next.taken[mission.id]++
			changed = true
		}
	}
	if !changed {
		return s
	}
	next.takesBelow = take.Position + 1
	return next
}

type Payment struct {
	Mission MissionID
	Reward  Reward
}

// Each mission done is paid once.
func (s Sheet) WithPayments(brief Brief) (Sheet, []Payment) {
	next := s.clone()
	var payments []Payment
	for _, mission := range brief.missions {
		if next.taken[mission.id] >= mission.goal && !next.paid.Contains(mission.id) {
			next.paid.Add(mission.id)
			payments = append(payments, Payment{Mission: mission.id, Reward: mission.reward})
		}
	}
	return next, payments
}
