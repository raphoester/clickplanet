// Package arenanight is a test alone: an arena battle from Fri 30 Oct 2026 21:00 to 23:00 in season 0, second by second, with
// scoring and planet answering late, and boots in the middle.
package arenanight_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_plan_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	scoringLate = 2 * time.Second
	planetLate  = 3 * time.Second
)

var (
	arenaStarts = time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	arenaEnds   = arenaStarts.Add(2 * time.Hour)
	arenaEntry  = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000032"))
	fought      = moments.BattleRoundOf(arenaEntry)
	seasonZero  = season.Snapshot{
		Number: 0, Length: 23*24*time.Hour + 23*time.Hour,
		Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3},
	}
)

// journal is what happened, in order: what the gamemaster sent, and what scoring and planet answered.
type journal struct {
	clock   *cptime.FixedClock
	entries []string
}

func (j *journal) note(format string, args ...any) {
	j.entries = append(j.entries, j.clock.Now().Format("15:04:05")+" "+fmt.Sprintf(format, args...))
}

func (j *journal) index(t *testing.T, entry string) int {
	t.Helper()
	i := slices.IndexFunc(j.entries, func(e string) bool { return e[9:] == entry })
	require.GreaterOrEqual(t, i, 0, "%q never happened: %v", entry, j.entries)
	return i
}

type answer struct {
	at   time.Time
	give func(ctx context.Context)
}

// late answers what it is told some seconds after.
type late struct {
	answers []answer
}

func (l *late) after(at time.Time, give func(ctx context.Context)) {
	l.answers = append(l.answers, answer{at: at, give: give})
}

func (l *late) deliver(ctx context.Context, now time.Time) {
	due := slices.DeleteFunc(slices.Clone(l.answers), func(a answer) bool { return a.at.After(now) })
	l.answers = slices.DeleteFunc(l.answers, func(a answer) bool { return !a.at.After(now) })
	for _, a := range due {
		a.give(ctx)
	}
}

type scoring struct {
	t       *testing.T
	journal *journal
	late    *late
	rounds  *hear_round_usecase.Hearing
	missed  *hear_missed_round_usecase.Hearing
	opened  *cpcolls.Set[moments.RoundID]
	down    error
}

func (s *scoring) Send(_ context.Context, command moments.Command) error {
	if s.down != nil {
		return s.down
	}
	switch command.Action {
	case moments.OpenRound:
		s.opened.Add(command.Round)
		s.journal.note("open round %s from nothing %t", command.Round, command.FromNothing)
	case moments.CloseRound:
		s.journal.note("close round %s", command.Round)
		opened := s.opened.Contains(command.Round)
		s.late.after(s.journal.clock.Now().Add(scoringLate), func(ctx context.Context) {
			if !opened {
				_, err := s.missed.Execute(ctx, moments.MissedRound{Round: command.Round, EndedAt: command.At})
				require.NoError(s.t, err)
				s.journal.note("missed %s", command.Round)
				return
			}
			_, err := s.rounds.Execute(ctx, moments.ScoredRound{Round: command.Round, Book: command.Book, EndedAt: command.At})
			require.NoError(s.t, err)
			s.journal.note("scored %s", command.Round)
		})
	case moments.OpenBook, moments.CloseBook:
	}
	return nil
}

// planet keeps each arena once, as its arena store does, and tells ArenaClosed once.
type planet struct {
	t       *testing.T
	journal *journal
	late    *late
	hear    *hear_arena_usecase.Hearing
	opened  *cpcolls.Set[moments.RoundID]
	closed  *cpcolls.Set[moments.RoundID]
}

func (p *planet) Arena(_ context.Context, set schedule.SetName) (bool, error) {
	return set == "arena-battle", nil
}

func (p *planet) Reset(_ context.Context, starts season.Window) error {
	p.journal.note("reset for season %d", starts.Number)
	return nil
}

func (p *planet) Send(_ context.Context, command moments.ArenaCommand) error {
	switch command.Action {
	case moments.OpenArena:
		if !p.opened.Contains(command.Arena) && !p.closed.Contains(command.Arena) {
			p.opened.Add(command.Arena)
			p.journal.note("blank the map")
		}
	case moments.CloseArena:
		p.late.after(p.journal.clock.Now().Add(planetLate), func(ctx context.Context) {
			if p.closed.Contains(command.Arena) {
				return
			}
			p.closed.Add(command.Arena)
			_, err := p.hear.Execute(ctx, command.Arena)
			require.NoError(p.t, err)
			p.journal.note("map back")
		})
	}
	return nil
}

type night struct {
	t       *testing.T
	clock   *cptime.FixedClock
	journal *journal
	late    *late
	planner moments.Planner
	kept    *inmemory_moment_store.Store
	judged  *inmemory_arena_store.Store
	outbox  *cpatomic.FakeOutbox
	scoring *scoring
	planet  *planet

	plan *play_plan_usecase.UseCase
	told int
}

func newNight(t *testing.T, at time.Time) *night {
	t.Helper()
	clock := cptime.NewFixedClock(at)
	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: arenaEntry, Set: "arena-battle", Kind: schedule.Battle, Description: "Arena battle.", StartsAt: arenaStarts,
		EndsAt: arenaEnds, Priority: 1, Times: 2, MadeAt: arenaStarts.Add(-24 * time.Hour),
	}.Restore()))
	kept, judged, outbox := inmemory_moment_store.New(), inmemory_arena_store.New(), &cpatomic.FakeOutbox{}
	planner := moments.NewPlanner(seasons, entries, kept, judged, season.Cutoff(21*time.Hour))
	notes, answers := &journal{clock: clock}, &late{}
	n := &night{
		t: t, clock: clock, journal: notes, late: answers, planner: planner, kept: kept, judged: judged, outbox: outbox,
		scoring: &scoring{
			t: t, journal: notes, late: answers, missed: hear_missed_round_usecase.New(kept),
			rounds: hear_round_usecase.New(planner, kept, outbox_moment_events.New(outbox), cpatomic.FakeTransactioner{}),
			opened: cpcolls.NewSet[moments.RoundID](),
		},
		planet: &planet{
			t: t, journal: notes, late: answers, hear: hear_arena_usecase.New(judged),
			opened: cpcolls.NewSet[moments.RoundID](), closed: cpcolls.NewSet[moments.RoundID](),
		},
	}
	n.boot()
	return n
}

// boot is a restart: what each runner sent is forgotten, what postgres keeps stays.
func (n *night) boot() {
	n.plan = play_plan_usecase.New(n.planner,
		play_resets_usecase.New(n.planet, n.clock),
		play_arenas_usecase.New(n.judged, n.planet, n.planet, n.clock, time.Minute),
		play_scoring_usecase.New(n.kept, n.scoring, n.clock, time.Minute),
		tell_starts_usecase.New(n.kept, outbox_moment_events.New(n.outbox), cpatomic.FakeTransactioner{}, n.clock))
}

func (n *night) tick() {
	n.t.Helper()
	n.late.deliver(n.t.Context(), n.clock.Now())
	require.NoError(n.t, n.plan.Execute(n.t.Context()))
	recorded := n.outbox.Recorded()
	for _, event := range recorded[n.told:] {
		if day, ok := event.(*gamemasterv1.DayStarted); ok {
			n.journal.note("day %d told", day.GetDay())
		}
	}
	n.told = len(recorded)
}

func (n *night) until(end time.Time, boots ...time.Time) {
	n.t.Helper()
	for n.clock.Now().Before(end) {
		n.clock.Advance(time.Second)
		if slices.ContainsFunc(boots, n.clock.Now().Equal) {
			n.boot()
		}
		n.tick()
	}
}

func TestAnArenaBattleBlanksTheMapAfterDay23IsScoredAndDay24WaitsForTheMapBack(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))
	n.journal.entries = nil

	n.until(arenaEnds.Add(time.Minute))

	assert.Equal(t, []string{
		"21:00:00 close round season-0/day-23",
		"21:00:00 open round " + string(fought) + " from nothing true",
		"21:00:02 scored season-0/day-23",
		"21:00:02 blank the map",
		"23:00:00 close round " + string(fought),
		"23:00:02 scored " + string(fought),
		"23:00:05 map back",
		"23:00:05 open round season-0/day-24 from nothing false",
		"23:00:05 day 24 told",
	}, n.journal.entries)
}

func TestBootsMidBattleAndWhileItWaitsKeepTheOrder(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))

	n.until(arenaEnds.Add(time.Minute), arenaStarts.Add(time.Second), arenaStarts.Add(time.Hour), arenaEnds.Add(3*time.Second))

	j := n.journal
	assert.Less(t, j.index(t, "scored season-0/day-23"), j.index(t, "blank the map"), "day 23 is scored on the map")
	assert.Less(t, j.index(t, "scored "+string(fought)), j.index(t, "map back"), "the arena battle is scored on its arena")
	assert.Less(t, j.index(t, "map back"), j.index(t, "open round season-0/day-24 from nothing false"))
	assert.Less(t, j.index(t, "map back"), j.index(t, "day 24 told"))
	assert.Equal(t, 1, j.count("blank the map"))
	assert.Equal(t, 1, j.count("map back"))
}

func TestAnArenaBattleWhoseRoundNeverOpenedClosesItsArenaAfterABootAndDay24Opens(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))
	n.journal.entries = nil

	n.clock.Advance(time.Second)
	n.scoring.down = errors.New("redis is down")
	require.Error(t, n.plan.Execute(t.Context()), "the battle is judged, and its round's open is refused")
	n.scoring.down = nil
	n.clock.Advance(arenaEnds.Add(time.Hour).Sub(n.clock.Now()))
	n.boot()
	n.until(arenaEnds.Add(time.Hour + time.Minute))

	assert.Equal(t, []string{
		"00:00:01 close round season-0/day-23",
		"00:00:01 close round " + string(fought),
		"00:00:03 scored season-0/day-23",
		"00:00:03 missed " + string(fought),
		"00:00:06 map back",
		"00:00:06 open round season-0/day-24 from nothing false",
		"00:00:06 day 24 told",
	}, n.journal.entries)
	assert.Zero(t, n.journal.count("blank the map"), "the battle ended before day 23 was scored: no arena opened")
}

func (j *journal) count(entry string) int {
	count := 0
	for _, e := range j.entries {
		if e[9:] == entry {
			count++
		}
	}
	return count
}
