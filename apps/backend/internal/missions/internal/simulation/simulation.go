package simulation

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

var ErrUnknownAct = errors.New("not an act of the ledger")

// A sizing under the name the report gives it.
type Named struct {
	Name   string
	Sizing flagmissions.Sizing
}

// Replays the acts of a season from a blank map, in the days told, and draws each flag's mission at its first take or
// shield broken of a day, as the game does, with each sizing: what it would have asked, and when the flag's players,
// playing as they did, would have done it. calendar is every day told, acts every act from the season's start.
func Run(calendar []days.Day, acts []Act, globe *Globe, sizings []Named) (Report, error) {
	run := newRun(calendar, globe, sizings)
	for _, act := range acts {
		if err := run.see(act); err != nil {
			return Report{}, err
		}
	}
	if err := run.close(); err != nil {
		return Report{}, err
	}
	return run.report(), nil
}

type flagDay struct {
	day     flagmissions.Day
	drawnAt time.Time
	share   float64
	doneAt  time.Time
}

type run struct {
	calendar []days.Day
	globe    *Globe
	sizings  []Named

	current   int
	today     []map[string]*flagDay
	yesterday []map[string]*flagDay
	// The last day that ended, and its flags' days: a battle may run between it and the next.
	last  int
	ended []map[string]*flagDay
	rows  [][]Row
}

func newRun(calendar []days.Day, globe *Globe, sizings []Named) *run {
	sorted := slices.Clone(calendar)
	slices.SortFunc(sorted, func(a, b days.Day) int { return a.Snapshot().StartsAt.Compare(b.Snapshot().StartsAt) })
	r := &run{calendar: sorted, globe: globe, sizings: sizings, current: -1, last: -1, rows: make([][]Row, len(sizings))}
	r.today, r.yesterday, r.ended = r.blank(), r.blank(), r.blank()
	return r
}

func (r *run) blank() []map[string]*flagDay {
	blank := make([]map[string]*flagDay, len(r.sizings))
	for n := range blank {
		blank[n] = make(map[string]*flagDay)
	}
	return blank
}

func (r *run) see(act Act) error {
	if err := r.enter(act.happened()); err != nil {
		return err
	}
	switch act := act.(type) {
	case Take:
		r.globe.Paint(act.Tile, act.Flag, act.Previous)
		if act.Account {
			r.count(act.Flag, act.At, func(day flagmissions.Day) (flagmissions.Day, []flagmissions.Done) {
				return day.WithTake(flagmissions.Take{
					Flag: act.Flag, Ground: r.globe.GroundOf(act.Tile), Position: act.Position, Index: act.Index, At: act.At,
				})
			})
		}
	case Strike:
		r.count(act.Flag, act.At, func(day flagmissions.Day) (flagmissions.Day, []flagmissions.Done) {
			return day.WithStrike(flagmissions.Strike{
				Flag: act.Flag, Owner: act.Owner, Position: act.Position, Index: act.Index, At: act.At,
			})
		})
	case Clear:
		r.globe.Paint(act.Tile, "", act.Previous)
	default:
		return fmt.Errorf("%w: %T", ErrUnknownAct, act)
	}
	return nil
}

// Moves to the day that runs at the time, closing the one before: a fact between two days counts nothing.
func (r *run) enter(at time.Time) error {
	if r.current >= 0 && r.calendar[r.current].Runs(at) {
		return nil
	}
	next := slices.IndexFunc(r.calendar, func(day days.Day) bool { return day.Runs(at) })
	if next == r.current {
		return nil
	}
	if err := r.close(); err != nil {
		return err
	}
	if r.current >= 0 {
		r.last, r.ended = r.current, r.today
	}
	if next >= 0 && r.last >= 0 && r.calendar[next].Follows(r.calendar[r.last]) {
		r.yesterday = r.ended
	} else {
		r.yesterday = r.blank()
	}
	r.today, r.current = r.blank(), next
	return nil
}

// The flag's day is drawn at its first fact, from the map then and its day before, as Briefing.Of draws it.
func (r *run) count(flag string, at time.Time, fact func(flagmissions.Day) (flagmissions.Day, []flagmissions.Done)) {
	if r.current < 0 {
		return
	}
	day := r.calendar[r.current]
	var world *missions.World
	for n, named := range r.sizings {
		kept, ok := r.today[n][flag]
		if !ok {
			if world == nil {
				drawn := r.globe.World()
				world = &drawn
			}
			var before flagmissions.Day
			if yesterday, ok := r.yesterday[n][flag]; ok {
				before = yesterday.day
			}
			kept = &flagDay{
				day:     flagmissions.Draw(day, flag, *world, before, named.Sizing),
				drawnAt: at,
				share:   r.globe.Share(flag),
			}
			r.today[n][flag] = kept
		}
		next, done := fact(kept.day)
		kept.day = next
		if len(done) > 0 {
			kept.doneAt = at
		}
	}
}

func (r *run) close() error {
	if r.current < 0 {
		return nil
	}
	day := r.calendar[r.current].Snapshot().ID
	for n, named := range r.sizings {
		for _, flag := range slices.Sorted(maps.Keys(r.today[n])) {
			kept := r.today[n][flag]
			var before flagmissions.DaySnapshot
			if yesterday, ok := r.yesterday[n][flag]; ok {
				before = yesterday.day.Snapshot()
			}
			row, err := rowOf(named.Name, day, flag, kept, before)
			if err != nil {
				return err
			}
			r.rows[n] = append(r.rows[n], row)
		}
	}
	return nil
}

func (r *run) report() Report {
	report := Report{Drift: r.globe.drift, Outside: r.globe.outside}
	for n, named := range r.sizings {
		report.Sizings = append(report.Sizings, SizingReport{Named: named, Rows: r.rows[n], Lines: linesOf(r.rows[n])})
	}
	return report
}
