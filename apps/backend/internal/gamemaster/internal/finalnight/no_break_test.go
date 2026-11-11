package finalnight_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/play_resets_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// scoringLands is how long scoring takes to close once the night starts: longer than the gamemaster's call, which is
// cut, and called again.
const scoringLands = 2 * time.Second

// errCut is a call the gamemaster's timeout cut: scoring goes on with it.
var errCut = errors.New("deadline exceeded")

type boundary struct {
	t       *testing.T
	clock   *cptime.FixedClock
	kept    *inmemory_moment_store.Store
	judged  *inmemory_arena_store.Store
	planner moments.Planner
	plan    *play_plan_usecase.UseCase
	maps    *play_maps_usecase.UseCase
	journal []string
	slow    time.Duration
	landsAt map[moments.Key]time.Time
	landed  []moments.Command
}

func newBoundary(t *testing.T, at time.Time) *boundary {
	t.Helper()
	seasons, kept, judged := inmemory_season_store.New(), inmemory_moment_store.New(), inmemory_arena_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 1, Length: 28 * 24 * time.Hour}.Restore()))
	b := &boundary{
		t: t, clock: cptime.NewFixedClock(at), kept: kept, judged: judged, landsAt: map[moments.Key]time.Time{},
		planner: moments.NewPlanner(seasons, inmemory_entry_store.New(), kept, judged, season.Cutoff(21*time.Hour)),
	}
	b.boot()
	return b
}

func (b *boundary) boot() {
	b.plan = play_plan_usecase.New(b.planner, judge_battles_usecase.New(b.judged, rulesOf{}, b.clock),
		play_scoring_usecase.New(b.kept, b, outbox_moment_events.New(&cpatomic.FakeOutbox{}), cpatomic.FakeTransactioner{},
			b.clock, time.Minute),
		tell_starts_usecase.New(b.kept, outbox_moment_events.New(&cpatomic.FakeOutbox{}), cpatomic.FakeTransactioner{}, b.clock))
	b.maps = play_maps_usecase.New(b.planner,
		play_resets_usecase.New(b.kept, b, b.clock, time.Minute),
		play_arenas_usecase.New(b.judged, &arenaCommands{}, b.clock, time.Minute))
}

func (b *boundary) note(format string, args ...any) {
	b.journal = append(b.journal, b.clock.Now().Format("15:04:05")+" "+fmt.Sprintf(format, args...))
}

func (b *boundary) Reset(_ context.Context, starts season.Window) error {
	b.note("reset for season %d", starts.Number)
	return nil
}

func (b *boundary) Ask(_ context.Context, command moments.Command) (moments.Answer, error) {
	if command.Action == moments.OpenBook || command.Action == moments.OpenRound {
		return moments.Answer{}, nil
	}
	if _, ok := b.landsAt[command.Key()]; !ok {
		b.landsAt[command.Key()] = b.clock.Now().Add(b.slow)
		subject := string(command.Book)
		if command.Action == moments.CloseRound {
			subject = string(command.Round)
		}
		b.note("%s %s", command.Action, subject)
	}
	if b.clock.Now().Before(b.landsAt[command.Key()]) {
		b.later(command)
		return moments.Answer{}, errCut
	}
	return moments.Answer{EndedAt: command.At}, nil
}

func (b *boundary) later(command moments.Command) {
	if !slices.Contains(b.landed, command) {
		b.landed = append(b.landed, command)
	}
}

// land notes what scoring finished, as it finishes it.
func (b *boundary) land() {
	for _, command := range b.landed {
		if b.clock.Now().Equal(b.landsAt[command.Key()]) {
			if command.Action == moments.CloseRound {
				b.note("scored %s", command.Round)
				continue
			}
			b.note("closed %s", command.Book)
		}
	}
}

func (b *boundary) until(end time.Time) {
	b.t.Helper()
	for b.clock.Now().Before(end) {
		b.clock.Advance(time.Second)
		b.land()
		if err := b.plan.Execute(b.t.Context()); !errors.Is(err, errCut) {
			require.NoError(b.t, err)
		}
		require.NoError(b.t, b.maps.Execute(b.t.Context()))
	}
}

func (b *boundary) index(entry string) int {
	b.t.Helper()
	i := slices.IndexFunc(b.journal, func(e string) bool { return e[9:] == entry })
	require.GreaterOrEqual(b.t, i, 0, "%q never happened: %v", entry, b.journal)
	return i
}

func TestSeasonOneRightAfterTheFinaleWipesTheMapOnlyOnceTheFinaleIsScored(t *testing.T) {
	b := newBoundary(t, seasonEnds.Add(-time.Minute))
	b.until(seasonEnds.Add(-time.Second))
	b.journal, b.slow = nil, scoringLands

	b.until(seasonEnds.Add(3 * time.Minute))

	assert.Equal(t, []string{
		"23:00:00 close round season-0/finale",
		"23:00:02 scored season-0/finale",
		"23:01:00 close book season-0",
		"23:01:02 closed season-0",
		"23:02:00 reset for season 1",
	}, b.journal, "each cut call is answered a minute later, and nothing goes before its answer")
}

func TestARestartAcrossTheSeasonsEndWipesTheMapOnlyOnceTheFinaleIsScored(t *testing.T) {
	b := newBoundary(t, seasonEnds.Add(-time.Minute))
	b.until(seasonEnds.Add(-time.Minute + time.Second))
	b.journal, b.slow = nil, scoringLands

	b.clock.Advance(30 * time.Minute)
	b.boot()
	b.until(seasonEnds.Add(33 * time.Minute))

	assert.Less(t, b.index("close round season-0/finale"), b.index("scored season-0/finale"))
	assert.Less(t, b.index("scored season-0/finale"), b.index("reset for season 1"), "the finale is scored on its map")
	assert.Less(t, b.index("closed season-0"), b.index("reset for season 1"))
}
