package flagmissions

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type DaySnapshot struct {
	// None when the map had nothing to aim at.
	Mission      *MissionSnapshot
	Progress     uint32
	Done         bool
	Takes        uint64
	TakesBelow   missions.TakeMark
	Strikes      uint64
	StrikesUntil time.Time
}

// A flag's day: its mission, and what all its players did, which sizes the next day's goal.
type Day struct {
	mission      *MissionSnapshot
	progress     uint32
	done         bool
	takes        uint64
	takesBelow   missions.TakeMark
	strikes      uint64
	strikesUntil time.Time
}

func DayOf(snapshot DaySnapshot) (Day, error) {
	day := Day{
		progress: snapshot.Progress, done: snapshot.Done, takes: snapshot.Takes, takesBelow: snapshot.TakesBelow,
		strikes: snapshot.Strikes, strikesUntil: snapshot.StrikesUntil,
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
		StrikesUntil: d.strikesUntil,
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
	next.takesBelow = missions.TakeMark{Position: take.Position, Index: take.Index + 1}
	if next.mission != nil {
		next.progress = next.mission.Order.advancedByTake(next.progress, take)
	}
	return next.finished()
}

// Strikes come in the order of their times: one at or before the mark was counted already.
func (d Day) WithStrike(strike Strike) (Day, []Done) {
	if !strike.At.After(d.strikesUntil) {
		return d, nil
	}
	next := d
	next.strikes++
	next.strikesUntil = strike.At
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
