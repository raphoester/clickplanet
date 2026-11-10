package finalnight_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/inmemory_arena_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/play_arenas_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
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

const scoringAnswers = 2 * time.Second

type boundary struct {
	t       *testing.T
	clock   *cptime.FixedClock
	kept    *inmemory_moment_store.Store
	planner moments.Planner
	plan    *play_plan_usecase.UseCase
	journal []string
	answers []answer
}

type answer struct {
	at   time.Time
	give func()
}

func newBoundary(t *testing.T, at time.Time) *boundary {
	t.Helper()
	seasons, kept := inmemory_season_store.New(), inmemory_moment_store.New()
	require.NoError(t, seasons.Save(t.Context(), seasonZero.Restore()))
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 1, Length: 28 * 24 * time.Hour}.Restore()))
	b := &boundary{
		t: t, clock: cptime.NewFixedClock(at), kept: kept,
		planner: moments.NewPlanner(seasons, inmemory_entry_store.New(), kept, inmemory_arena_store.New(), season.Cutoff(21*time.Hour)),
	}
	b.boot()
	return b
}

func (b *boundary) boot() {
	b.plan = play_plan_usecase.New(b.planner,
		play_resets_usecase.New(b, b.clock),
		play_arenas_usecase.New(inmemory_arena_store.New(), rulesOf{}, &arenaCommands{}, b.clock, time.Minute),
		play_scoring_usecase.New(b.kept, b, b.clock, time.Minute),
		tell_starts_usecase.New(b.kept, outbox_moment_events.New(&cpatomic.FakeOutbox{}), cpatomic.FakeTransactioner{}, b.clock))
}

func (b *boundary) note(format string, args ...any) {
	b.journal = append(b.journal, b.clock.Now().Format("15:04:05")+" "+fmt.Sprintf(format, args...))
}

func (b *boundary) Reset(_ context.Context, starts season.Window) error {
	b.note("reset for season %d", starts.Number)
	return nil
}

func (b *boundary) Send(ctx context.Context, command moments.Command) error {
	switch command.Action {
	case moments.CloseRound:
		b.note("close round %s", command.Round)
		b.later(func() {
			_, err := b.kept.KeepRound(ctx, moments.ScoredRound{Round: command.Round, Book: command.Book, EndedAt: command.At})
			require.NoError(b.t, err)
			b.note("scored %s", command.Round)
		})
	case moments.CloseBook:
		b.note("close book %s", command.Book)
		b.later(func() {
			_, err := b.kept.KeepBook(ctx, moments.ClosedBook{Book: command.Book, EndedAt: command.At})
			require.NoError(b.t, err)
			b.note("closed %s", command.Book)
		})
	case moments.OpenBook, moments.OpenRound:
	}
	return nil
}

func (b *boundary) later(give func()) {
	b.answers = append(b.answers, answer{at: b.clock.Now().Add(scoringAnswers), give: give})
}

func (b *boundary) until(end time.Time) {
	b.t.Helper()
	for b.clock.Now().Before(end) {
		b.clock.Advance(time.Second)
		due := slices.DeleteFunc(slices.Clone(b.answers), func(a answer) bool { return a.at.After(b.clock.Now()) })
		b.answers = slices.DeleteFunc(b.answers, func(a answer) bool { return !a.at.After(b.clock.Now()) })
		for _, a := range due {
			a.give()
		}
		require.NoError(b.t, b.plan.Execute(b.t.Context()))
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
	b.journal = nil

	b.until(seasonEnds.Add(time.Minute))

	assert.Equal(t, []string{
		"23:00:00 close round season-0/finale",
		"23:00:00 close book season-0",
		"23:00:02 scored season-0/finale",
		"23:00:02 closed season-0",
		"23:00:02 reset for season 1",
	}, b.journal)
}

func TestARestartAcrossTheSeasonsEndWipesTheMapOnlyOnceTheFinaleIsScored(t *testing.T) {
	b := newBoundary(t, seasonEnds.Add(-time.Minute))
	b.until(seasonEnds.Add(-time.Minute + time.Second))
	b.journal = nil

	b.clock.Advance(30 * time.Minute)
	b.boot()
	b.until(seasonEnds.Add(31 * time.Minute))

	assert.Less(t, b.index("close round season-0/finale"), b.index("scored season-0/finale"))
	assert.Less(t, b.index("scored season-0/finale"), b.index("reset for season 1"), "the finale is scored on its map")
	assert.Less(t, b.index("closed season-0"), b.index("reset for season 1"))
}
