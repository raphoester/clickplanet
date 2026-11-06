package simulation

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

type Report struct {
	Sizings []SizingReport
	// Acts whose tile wore another flag than the replay said: a reassign, a paint or a revert the ledger never saw.
	Drift uint64
	// Acts on a tile past the map.
	Outside uint64
}

type SizingReport struct {
	Named
	// Every flag's day, by day then flag.
	Rows []Row
	// The rows by the flag's size, by kind, and all of them.
	Lines []Line
}

type Kind string

const (
	NoMission Kind = "none"
	Conquest  Kind = "conquest"
	Shields   Kind = "shields"
)

// A flag's day: the mission it was drawn, and what its players did, as they played it.
type Row struct {
	Sizing string
	Day    days.DayID
	Flag   string
	// The part of the map the flag held at the draw.
	Share  float64
	Kind   Kind
	Points uint32
	// The ground of a conquest, the flag whose shields to break.
	Target string
	Goal   uint32
	// What the flag's players did the day before, and that day.
	TakesBefore, StrikesBefore uint64
	Takes, Strikes             uint64
	Progress                   uint32
	Done                       bool
	// From the draw, the flag's first fact of the day, to the fact that finished it.
	ToDone time.Duration
}

func rowOf(name string, day days.DayID, flag string, kept *flagDay, before flagmissions.DaySnapshot) (Row, error) {
	snapshot := kept.day.Snapshot()
	row := Row{
		Sizing: name, Day: day, Flag: flag, Share: kept.share, Kind: NoMission, TakesBefore: before.Takes,
		StrikesBefore: before.Strikes, Takes: snapshot.Takes, Strikes: snapshot.Strikes,
	}
	status, ok := kept.day.Status()
	if !ok {
		return row, nil
	}
	switch order := status.Mission.Order.(type) {
	case flagmissions.TakeIn:
		row.Kind, row.Target, row.Goal = Conquest, order.Ground, order.Goal
	case flagmissions.BreakShields:
		row.Kind, row.Target, row.Goal = Shields, order.Flag, order.Goal
	default:
		return Row{}, fmt.Errorf("%w: %T", flagmissions.ErrNoOrder, order)
	}
	row.Points, row.Progress, row.Done = status.Mission.Points, status.Progress, status.Done
	if row.Done {
		row.ToDone = kept.doneAt.Sub(kept.drawnAt)
	}
	return row, nil
}

// The flag days with a mission of a group.
type Line struct {
	Group    string
	FlagDays int
	Done     int
	// Of the missions done.
	MedianToDone time.Duration
	MedianGoal   uint32
	// The goal beside all the flag's players did that day, tiles taken and shields broken.
	MedianGoalShare float64
	PointsOffered   uint64
	PointsWon       uint64
}

const (
	bigFlag   = 0.05
	smallFlag = 0.005
)

func linesOf(rows []Row) []Line {
	groups := []struct {
		name string
		in   func(Row) bool
	}{
		{"flags of 5% of the map or more", func(row Row) bool { return row.Share >= bigFlag }},
		{"flags of 0.5% to 5%", func(row Row) bool { return row.Share >= smallFlag && row.Share < bigFlag }},
		{"flags under 0.5%", func(row Row) bool { return row.Share < smallFlag }},
		{"conquests", func(row Row) bool { return row.Kind == Conquest }},
		{"shields to break", func(row Row) bool { return row.Kind == Shields }},
		{"every flag", func(Row) bool { return true }},
	}
	lines := make([]Line, 0, len(groups))
	for _, group := range groups {
		lines = append(lines, lineOf(group.name, slices.DeleteFunc(slices.Clone(rows), func(row Row) bool {
			return row.Kind == NoMission || !group.in(row)
		})))
	}
	return lines
}

func lineOf(group string, rows []Row) Line {
	line := Line{Group: group, FlagDays: len(rows)}
	var (
		toDone []time.Duration
		goals  = make([]uint32, 0, len(rows))
		shares = make([]float64, 0, len(rows))
	)
	for _, row := range rows {
		line.PointsOffered += uint64(row.Points)
		goals = append(goals, row.Goal)
		if acts := row.Takes + row.Strikes; acts > 0 {
			shares = append(shares, float64(row.Goal)/float64(acts))
		}
		if row.Done {
			line.Done++
			line.PointsWon += uint64(row.Points)
			toDone = append(toDone, row.ToDone)
		}
	}
	line.MedianToDone, line.MedianGoal, line.MedianGoalShare = median(toDone), median(goals), median(shares)
	return line
}

func median[T cmp.Ordered](values []T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	sorted := slices.Sorted(slices.Values(values))
	return sorted[len(sorted)/2]
}
