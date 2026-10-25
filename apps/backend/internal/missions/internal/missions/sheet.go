package missions

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var (
	ErrNotDone   = errors.New("the mission is not done")
	ErrCollected = errors.New("the mission was collected already")
)

type SheetSnapshot struct {
	Flag       string
	Taken      map[Slot]uint32
	Collected  *cpcolls.Set[Slot]
	TakesBelow uint64
}

type Sheet struct {
	flag       string
	taken      map[Slot]uint32
	collected  *cpcolls.Set[Slot]
	takesBelow uint64
}

func SheetOf(snapshot SheetSnapshot) Sheet {
	sheet := Sheet{flag: snapshot.Flag, taken: map[Slot]uint32{}, collected: cpcolls.NewSet[Slot](), takesBelow: snapshot.TakesBelow}
	for slot, taken := range snapshot.Taken {
		sheet.taken[slot] = taken
	}
	if snapshot.Collected != nil {
		sheet.collected.AddSet(snapshot.Collected)
	}
	return sheet
}

func (s Sheet) clone() Sheet {
	return SheetOf(SheetSnapshot{Flag: s.flag, Taken: s.taken, Collected: s.collected, TakesBelow: s.takesBelow})
}

func (s Sheet) Snapshot() SheetSnapshot {
	copied := s.clone()
	return SheetSnapshot{Flag: copied.flag, Taken: copied.taken, Collected: copied.collected, TakesBelow: copied.takesBelow}
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
		if mission.Counts(take) && next.taken[mission.slot] < mission.goal {
			next.taken[mission.slot]++
			changed = true
		}
	}
	if !changed {
		return s
	}
	next.takesBelow = take.Position + 1
	return next
}

func (s Sheet) CollectError(mission Mission) error {
	switch {
	case s.collected.Contains(mission.slot):
		return ErrCollected
	case s.taken[mission.slot] < mission.goal:
		return ErrNotDone
	}
	return nil
}
