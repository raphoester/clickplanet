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
	Flag      string
	Taken     map[Slot]uint32
	Collected *cpcolls.Set[Slot]
}

type Sheet struct {
	flag      string
	taken     map[Slot]uint32
	collected *cpcolls.Set[Slot]
}

func SheetOf(snapshot SheetSnapshot) Sheet {
	sheet := Sheet{flag: snapshot.Flag, taken: map[Slot]uint32{}, collected: cpcolls.NewSet[Slot]()}
	for slot, taken := range snapshot.Taken {
		sheet.taken[slot] = taken
	}
	if snapshot.Collected != nil {
		sheet.collected.AddSet(snapshot.Collected)
	}
	return sheet
}

func (s Sheet) clone() Sheet {
	return SheetOf(SheetSnapshot{Flag: s.flag, Taken: s.taken, Collected: s.collected})
}

func (s Sheet) Snapshot() SheetSnapshot {
	copied := s.clone()
	return SheetSnapshot{Flag: copied.flag, Taken: copied.taken, Collected: copied.collected}
}

func (s Sheet) Enlisted() bool { return s.flag != "" }

// The brief is the take's flag's: a sheet already enlisted for another flag ignores it.
func (s Sheet) WithTake(take Take, brief Brief) Sheet {
	s = s.clone()
	if s.flag == "" {
		s.flag = take.Flag
	}
	if take.Flag != s.flag {
		return s
	}

	for _, mission := range brief.missions {
		if mission.Counts(take) && s.taken[mission.slot] < mission.goal {
			s.taken[mission.slot]++
		}
	}
	return s
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
