package flagmissions

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type DaySnapshot struct {
	// None when the map had nothing to aim at.
	Mission      *MissionSnapshot
	Progress     uint32
	Done         bool
	Takes        uint64
	TakesBelow   missions.LedgerMark
	Strikes      uint64
	StrikesBelow missions.LedgerMark
}

// A flag's day: its mission, and what all its players did, which sizes the next day's goal.
type Day struct {
	mission      *MissionSnapshot
	progress     uint32
	done         bool
	takes        uint64
	takesBelow   missions.LedgerMark
	strikes      uint64
	strikesBelow missions.LedgerMark
}

func DayOf(snapshot DaySnapshot) (Day, error) {
	day := Day{
		progress: snapshot.Progress, done: snapshot.Done, takes: snapshot.Takes, takesBelow: snapshot.TakesBelow,
		strikes: snapshot.Strikes, strikesBelow: snapshot.StrikesBelow,
	}
	if snapshot.Mission != nil {
		if snapshot.Mission.Order == nil {
			return Day{}, ErrNoOrder
		}
		mission := *snapshot.Mission
		day.mission = &mission
	}
	return day, nil
}

func (d Day) Snapshot() DaySnapshot {
	snapshot := DaySnapshot{
		Progress: d.progress, Done: d.done, Takes: d.takes, TakesBelow: d.takesBelow, Strikes: d.strikes,
		StrikesBelow: d.strikesBelow,
	}
	if d.mission != nil {
		mission := *d.mission
		snapshot.Mission = &mission
	}
	return snapshot
}

// The mission the flag finished, and what it is worth.
type Done struct {
	Mission MissionID
	Points  uint32
}

// A take can be told twice: one below the mark was counted already. Takes come in ledger order, and the takes of one
// spread or enclose, which share a position, in index order.
func (d Day) WithTake(take Take) (Day, []Done) {
	if take.Position < d.takesBelow.Position || take.Position == d.takesBelow.Position && take.Index < d.takesBelow.Index {
		return d, nil
	}
	next := d
	next.takes++
	next.takesBelow = missions.LedgerMark{Position: take.Position, Index: take.Index + 1}
	if next.mission != nil {
		next.progress = next.mission.Order.advancedByTake(next.progress, take)
	}
	return next.finished()
}

// Strikes come in ledger order, as takes do: one below the mark was counted already. A bomb strikes its own flag's
// shields too, and they are no shield broken.
func (d Day) WithStrike(strike Strike) (Day, []Done) {
	if strike.Position < d.strikesBelow.Position ||
		strike.Position == d.strikesBelow.Position && strike.Index < d.strikesBelow.Index {
		return d, nil
	}
	next := d
	next.strikesBelow = missions.LedgerMark{Position: strike.Position, Index: strike.Index + 1}
	if strike.Owner == strike.Flag {
		return next, nil
	}
	next.strikes++
	if next.mission != nil {
		next.progress = next.mission.Order.advancedByStrike(next.progress, strike)
	}
	return next.finished()
}

func (d Day) finished() (Day, []Done) {
	if d.mission == nil || d.done || !d.mission.Order.done(d.progress) {
		return d, nil
	}
	d.done = true
	return d, []Done{{Mission: d.mission.ID, Points: d.mission.Points}}
}

type Status struct {
	Mission  MissionSnapshot
	Progress uint32
	Done     bool
}

// How far the flag took its mission; false when the day has none.
func (d Day) Status() (Status, bool) {
	if d.mission == nil {
		return Status{}, false
	}
	return Status{Mission: *d.mission, Progress: d.progress, Done: d.done}, true
}
