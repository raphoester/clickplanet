package missions

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type SheetSnapshot struct {
	Flag       string
	Progress   map[MissionID]uint32
	Paid       *cpcolls.Set[MissionID]
	TakesBelow uint64
}

type Sheet struct {
	flag       string
	progress   map[MissionID]uint32
	paid       *cpcolls.Set[MissionID]
	takesBelow uint64
}

func SheetOf(snapshot SheetSnapshot) Sheet {
	sheet := Sheet{flag: snapshot.Flag, progress: map[MissionID]uint32{}, paid: cpcolls.NewSet[MissionID](), takesBelow: snapshot.TakesBelow}
	for id, progress := range snapshot.Progress {
		sheet.progress[id] = progress
	}
	if snapshot.Paid != nil {
		sheet.paid.AddSet(snapshot.Paid)
	}
	return sheet
}

func (s Sheet) clone() Sheet {
	return SheetOf(SheetSnapshot{Flag: s.flag, Progress: s.progress, Paid: s.paid, TakesBelow: s.takesBelow})
}

func (s Sheet) Snapshot() SheetSnapshot {
	copied := s.clone()
	return SheetSnapshot{Flag: copied.flag, Progress: copied.progress, Paid: copied.paid, TakesBelow: copied.takesBelow}
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
		if advanced := mission.order.advanced(next.progress[mission.id], take); advanced != next.progress[mission.id] {
			next.progress[mission.id] = advanced
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
		if mission.order.done(next.progress[mission.id]) && !next.paid.Contains(mission.id) {
			next.paid.Add(mission.id)
			payments = append(payments, Payment{Mission: mission.id, Reward: mission.reward})
		}
	}
	return next, payments
}

type MissionStatus struct {
	Mission  MissionSnapshot
	Progress uint32
	Done     bool
	Paid     bool
}

// How far the sheet took each mission of the brief, in the brief's order.
func (s Sheet) Statuses(brief Brief) []MissionStatus {
	statuses := make([]MissionStatus, 0, len(brief.missions))
	for _, mission := range brief.missions {
		statuses = append(statuses, MissionStatus{
			Mission:  mission.Snapshot(),
			Progress: s.progress[mission.id],
			Done:     mission.order.done(s.progress[mission.id]),
			Paid:     s.paid.Contains(mission.id),
		})
	}
	return statuses
}
