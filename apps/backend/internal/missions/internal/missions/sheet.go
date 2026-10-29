package missions

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type SheetSnapshot struct {
	Progress map[MissionID]uint32
	Paid     *cpcolls.Set[MissionID]
	Counted  Counted
}

type Sheet struct {
	progress map[MissionID]uint32
	paid     *cpcolls.Set[MissionID]
	counted  Counted
}

func SheetOf(snapshot SheetSnapshot) Sheet {
	sheet := Sheet{progress: map[MissionID]uint32{}, paid: cpcolls.NewSet[MissionID](), counted: snapshot.Counted}
	for id, progress := range snapshot.Progress {
		sheet.progress[id] = progress
	}
	if snapshot.Paid != nil {
		sheet.paid.AddSet(snapshot.Paid)
	}
	return sheet
}

func (s Sheet) clone() Sheet {
	return SheetOf(SheetSnapshot{Progress: s.progress, Paid: s.paid, Counted: s.counted})
}

func (s Sheet) Snapshot() SheetSnapshot {
	copied := s.clone()
	return SheetSnapshot{Progress: copied.progress, Paid: copied.paid, Counted: copied.counted}
}

// A fact that moves no mission leaves the marks, so nothing is written.
func (s Sheet) WithFact(fact Fact, brief Brief) Sheet {
	counted, fresh := fact.counted(s.counted)
	if !fresh {
		return s
	}

	next := s.clone()
	changed := false
	for _, mission := range brief.missions {
		if advanced := mission.order.advanced(next.progress[mission.id], fact); advanced != next.progress[mission.id] {
			next.progress[mission.id] = advanced
			changed = true
		}
	}
	if !changed {
		return s
	}
	next.counted = counted
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
