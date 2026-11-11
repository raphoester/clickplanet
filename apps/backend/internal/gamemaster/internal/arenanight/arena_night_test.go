// Package arenanight is a test alone: an arena battle from Fri 30 Oct 2026 21:00 to 23:00 in season 0, second by second, with
// planet slower than the gamemaster's calls, and boots in the middle.
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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/judge_battles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_maps_usecase"
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

var (
	// errCut is a call the gamemaster's timeout cut: the receiver goes on with it.
	errCut  = errors.New("deadline exceeded")
	errDown = errors.New("scoring is down")
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

// journal is what happened, in order: what the gamemaster called, and what scoring and planet did.
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

func (j *journal) count(entry string) int {
	count := 0
	for _, e := range j.entries {
		if e[9:] == entry {
			count++
		}
	}
	return count
}

// scoring answers each call at once: a round it opened is scored, one it never opened missed.
type scoring struct {
	journal *journal
	opened  *cpcolls.Set[moments.RoundID]
	down    error
}

func (s *scoring) Ask(_ context.Context, command moments.Command) (moments.Answer, error) {
	if s.down != nil {
		return moments.Answer{}, s.down
	}
	switch command.Action {
	case moments.OpenRound:
		s.opened.Add(command.Round)
		s.journal.note("open round %s from nothing %t", command.Round, command.FromNothing)
	case moments.CloseRound:
		s.journal.note("close round %s", command.Round)
		if !s.opened.Contains(command.Round) {
			s.journal.note("missed %s", command.Round)
			return moments.Answer{Missed: true, EndedAt: command.At}, nil
		}
		s.journal.note("scored %s", command.Round)
	case moments.OpenBook, moments.CloseBook:
	}
	return moments.Answer{EndedAt: command.At}, nil
}

// planet swaps the map `slow` after it is first asked: a call before that is cut, and the swap lands all the same, as
// planet's runs detached from the call. It keeps each arena once, and refuses an open that lands at or after its end.
type planet struct {
	journal *journal
	slow    time.Duration
	asked   []moments.ArenaCommand
	landsAt map[moments.ArenaKey]time.Time
	answers map[moments.ArenaKey]error
	closed  *cpcolls.Set[moments.RoundID]
}

func (p *planet) Arena(_ context.Context, set schedule.SetName) (bool, error) {
	return set == "arena-battle", nil
}

func (p *planet) Ask(_ context.Context, command moments.ArenaCommand) error {
	now := p.journal.clock.Now()
	if _, ok := p.landsAt[command.Key()]; !ok {
		p.landsAt[command.Key()] = now.Add(p.slow)
		p.asked = append(p.asked, command)
	}
	p.land(now)
	answer, landed := p.answers[command.Key()]
	if !landed {
		return errCut
	}
	return answer
}

func (p *planet) land(now time.Time) {
	for _, command := range p.asked {
		if _, landed := p.answers[command.Key()]; landed || now.Before(p.landsAt[command.Key()]) {
			continue
		}
		p.answers[command.Key()] = p.swap(command, now)
	}
}

func (p *planet) swap(command moments.ArenaCommand, now time.Time) error {
	switch command.Action {
	case moments.OpenArena:
		if p.closed.Contains(command.Arena) || !now.Before(command.EndsAt) {
			p.journal.note("arena refused")
			return fmt.Errorf("%w: arena %s ended", moments.ErrRefused, command.Arena)
		}
		p.journal.note("blank the map")
	case moments.CloseArena:
		p.closed.Add(command.Arena)
		p.journal.note("map back")
	}
	return nil
}

type resets struct{}

func (resets) Reset(context.Context, season.Window) error { return nil }

type night struct {
	t       *testing.T
	clock   *cptime.FixedClock
	journal *journal
	planner moments.Planner
	kept    *inmemory_moment_store.Store
	judged  *inmemory_arena_store.Store
	outbox  *cpatomic.FakeOutbox
	scoring *scoring
	planet  *planet

	plan *play_plan_usecase.UseCase
	maps *play_maps_usecase.UseCase
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
	kept, judged := inmemory_moment_store.New(), inmemory_arena_store.New()
	notes := &journal{clock: clock}
	n := &night{
		t: t, clock: clock, journal: notes, kept: kept, judged: judged, outbox: &cpatomic.FakeOutbox{},
		planner: moments.NewPlanner(seasons, entries, kept, judged, season.Cutoff(21*time.Hour)),
		scoring: &scoring{journal: notes, opened: cpcolls.NewSet[moments.RoundID]()},
		planet: &planet{
			journal: notes, landsAt: map[moments.ArenaKey]time.Time{}, answers: map[moments.ArenaKey]error{},
			closed: cpcolls.NewSet[moments.RoundID](),
		},
	}
	n.boot()
	return n
}

// boot is a restart: what each runner knew of its calls is forgotten, what postgres keeps stays.
func (n *night) boot() {
	told := outbox_moment_events.New(n.outbox)
	n.plan = play_plan_usecase.New(n.planner, judge_battles_usecase.New(n.judged, n.planet, n.clock),
		play_scoring_usecase.New(n.kept, n.scoring, told, cpatomic.FakeTransactioner{}, n.clock, time.Minute),
		tell_starts_usecase.New(n.kept, told, cpatomic.FakeTransactioner{}, n.clock))
	n.maps = play_maps_usecase.New(n.planner,
		play_resets_usecase.New(n.kept, resets{}, n.clock, time.Minute),
		play_arenas_usecase.New(n.judged, n.planet, n.clock, time.Minute))
}

// tick is a second: what planet runs lands, then each runner plays the plan it reads.
func (n *night) tick() {
	n.t.Helper()
	n.planet.land(n.clock.Now())
	for _, runner := range []play_plan_usecase.Executor{n.plan, n.maps} {
		if err := runner.Execute(n.t.Context()); !errors.Is(err, errCut) && !errors.Is(err, errDown) {
			require.NoError(n.t, err)
		}
	}
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
		"21:00:00 scored season-0/day-23",
		"21:00:00 blank the map",
		"21:00:01 open round " + string(fought) + " from nothing true",
		"23:00:00 close round " + string(fought),
		"23:00:00 scored " + string(fought),
		"23:00:00 map back",
		"23:00:01 open round season-0/day-24 from nothing false",
		"23:00:01 day 24 told",
	}, n.journal.entries)
}

func TestASlowPlanetIsCalledAgainAMinuteLaterAndNothingGoesBeforeItsAnswer(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))
	n.journal.entries = nil
	n.planet.slow = 3 * time.Second

	n.until(arenaEnds.Add(3 * time.Minute))

	assert.Equal(t, []string{
		"21:00:00 close round season-0/day-23",
		"21:00:00 scored season-0/day-23",
		"21:00:03 blank the map",
		"21:01:01 open round " + string(fought) + " from nothing true",
		"23:00:00 close round " + string(fought),
		"23:00:00 scored " + string(fought),
		"23:00:03 map back",
		"23:01:01 open round season-0/day-24 from nothing false",
		"23:01:01 day 24 told",
	}, n.journal.entries, "the call cut at 21:00:00 is answered at 21:01:00, and the battle's round opens on the blank map after")
}

func TestBootsMidBattleAndWhileItWaitsKeepTheOrder(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))
	n.planet.slow = 3 * time.Second

	n.until(arenaEnds.Add(3*time.Minute), arenaStarts.Add(time.Second), arenaStarts.Add(time.Hour), arenaEnds.Add(2*time.Second))

	j := n.journal
	assert.Less(t, j.index(t, "scored season-0/day-23"), j.index(t, "blank the map"), "day 23 is scored on the map")
	assert.Less(t, j.index(t, "blank the map"), j.index(t, "open round "+string(fought)+" from nothing true"),
		"the arena battle counts on its arena")
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
	n.scoring.down = errDown
	n.tick()
	n.scoring.down = nil
	n.clock.Advance(arenaEnds.Add(time.Hour).Sub(n.clock.Now()))
	n.boot()
	n.until(arenaEnds.Add(time.Hour + time.Minute))

	assert.Equal(t, []string{
		"00:00:01 close round season-0/day-23",
		"00:00:01 scored season-0/day-23",
		"00:00:01 close round " + string(fought),
		"00:00:01 missed " + string(fought),
		"00:00:01 map back",
		"00:00:02 open round season-0/day-24 from nothing false",
		"00:00:02 day 24 told",
	}, n.journal.entries)
	assert.Zero(t, n.journal.count("blank the map"), "the battle ended before day 23 was scored: no arena opened")
}

func TestAnArenaPlanetRefusesAtItsEndNeverOpensItsRoundAndDay24Opens(t *testing.T) {
	n := newNight(t, arenaStarts.Add(-time.Minute))
	n.until(arenaStarts.Add(-time.Second))
	n.journal.entries = nil
	n.planet.slow = arenaEnds.Sub(arenaStarts)
	n.until(arenaStarts)
	n.planet.slow = 0

	n.until(arenaEnds.Add(time.Minute))

	assert.Equal(t, []string{
		"21:00:00 close round season-0/day-23",
		"21:00:00 scored season-0/day-23",
		"23:00:00 arena refused",
		"23:00:00 close round " + string(fought),
		"23:00:00 missed " + string(fought),
		"23:00:00 map back",
		"23:00:01 open round season-0/day-24 from nothing false",
		"23:00:01 day 24 told",
	}, n.journal.entries, "the battle never counts from nothing on the world's map")
}
