package play_scoring_usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/play_scoring_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	day    = 24 * time.Hour
	cutoff = season.Cutoff(21 * time.Hour)
	book   = moments.BookID("season-0")
)

var (
	finale         = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	zero           = season.Snapshot{
		Number: 0, Length: 23*day + 23*time.Hour,
		Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3},
	}
	battle = moments.RoundID("season-0/finale")
	rush   = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))
)

type commanded struct {
	sent []moments.Command
	err  error
}

func (c *commanded) Send(_ context.Context, command moments.Command) error {
	if c.err != nil {
		return c.err
	}
	c.sent = append(c.sent, command)
	return nil
}

type harness struct {
	t        *testing.T
	clock    *cptime.FixedClock
	seasons  *inmemory_season_store.Store
	entries  *inmemory_entry_store.Store
	opened   *inmemory_moment_store.Store
	commands *commanded
	useCase  *play_scoring_usecase.UseCase
}

// Season 0 runs from its origin to 31 Oct 23:00, and its finale, the Final Battle, is rush × 3 from 21:00 to 23:00.
func newHarness(t *testing.T, at time.Time) *harness {
	t.Helper()

	h := &harness{
		t: t, clock: cptime.NewFixedClock(at), seasons: inmemory_season_store.New(), entries: inmemory_entry_store.New(),
		opened: inmemory_moment_store.New(),
	}
	require.NoError(t, h.seasons.Save(t.Context(), zero.Restore()))
	h.boot()
	return h
}

// Season 0 has no finale, and a battle entry, rush × 3, runs from 21:00 to 23:00.
func (h *harness) noFinale() {
	h.t.Helper()
	plain := zero
	plain.Finale = season.Finale{}
	require.NoError(h.t, h.seasons.Save(h.t.Context(), plain.Restore()))
	require.NoError(h.t, h.entries.Save(h.t.Context(), schedule.EntrySnapshot{
		ID: rush, Set: "rush", Kind: schedule.Battle, Description: "Triple points.", StartsAt: finale, EndsAt: seasonZeroEnds,
		Priority: 1, Times: 3, MadeAt: finale.Add(-day),
	}.Restore()))
}

func (h *harness) boot() {
	h.commands = &commanded{}
	h.useCase = play_scoring_usecase.New(moments.NewPlanner(h.seasons, h.entries, h.opened, cutoff), h.opened, h.commands, h.clock)
}

func (h *harness) at(at time.Time) {
	h.clock.Advance(at.Sub(h.clock.Now()))
}

func (h *harness) execute() []moments.Command {
	h.t.Helper()
	sent, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return sent
}

func dayRound(n int) moments.RoundID { return moments.RoundID(fmt.Sprintf("season-0/day-%d", n)) }

func dayEnds(n int) time.Time {
	return time.Date(2026, 10, 8, 21, 0, 0, 0, time.UTC).Add(time.Duration(n-1) * day)
}

func closeDay(n int) moments.Command {
	return moments.Command{Action: moments.CloseRound, Book: book, Round: dayRound(n), At: dayEnds(n)}
}

func TestTheFinalBattleNightIsCommandedInOrderAndOnce(t *testing.T) {
	h := newHarness(t, finale.Add(-time.Second))
	booted := h.execute()
	require.NotEmpty(t, booted)
	assert.Equal(t, moments.Command{Action: moments.OpenRound, Book: book, Round: dayRound(24), Times: 1, At: dayEnds(23)},
		booted[len(booted)-1], "the boot opens day 24, which runs")

	sentAt := map[time.Time][]moments.Command{}
	for !h.clock.Now().After(seasonZeroEnds.Add(time.Minute)) {
		h.clock.Advance(time.Second)
		if sent := h.execute(); len(sent) > 0 {
			sentAt[h.clock.Now()] = sent
		}
	}

	assert.Equal(t, map[time.Time][]moments.Command{
		finale: {
			closeDay(24),
			{Action: moments.OpenRound, Book: book, Round: battle, Times: 3, At: finale},
		},
		seasonZeroEnds: {
			{Action: moments.CloseRound, Book: book, Round: battle, At: seasonZeroEnds},
			{Action: moments.CloseBook, Book: book, Tiebreak: battle, At: seasonZeroEnds},
		},
	}, sentAt, "no day 25 runs from 21:00 to 23:00, and nothing is sent in between")

	h.at(seasonZeroEnds.Add(5 * time.Minute))
	h.boot()
	rebooted := h.execute()

	closes := make([]moments.Command, 0, 26)
	for n := 1; n <= 24; n++ {
		closes = append(closes, closeDay(n))
	}
	closes = append(closes,
		moments.Command{Action: moments.CloseRound, Book: book, Round: battle, At: seasonZeroEnds},
		moments.Command{Action: moments.CloseBook, Book: book, Tiebreak: battle, At: seasonZeroEnds},
	)
	assert.Equal(t, closes, rebooted, "a boot after the end closes what ended again, and opens nothing")
}

func TestABootMidDayOpensTheBookAndTheRunningDay(t *testing.T) {
	h := newHarness(t, time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))

	sent := h.execute()

	expected := []moments.Command{{Action: moments.OpenBook, Book: book, At: season.Origin}}
	for n := 1; n <= 12; n++ {
		expected = append(expected, closeDay(n))
	}
	expected = append(expected, moments.Command{Action: moments.OpenRound, Book: book, Round: dayRound(13), Times: 1, At: dayEnds(12)})
	assert.Equal(t, expected, sent)
	assert.Empty(t, h.execute(), "each is sent once")
}

func TestAMovedSeasonEndMovesItsBookItsDaysAndItsFinale(t *testing.T) {
	h := newHarness(t, finale.Add(-day))
	h.execute()
	moved := zero
	moved.Length += 7 * day
	require.NoError(t, h.seasons.Save(t.Context(), moved.Restore()))

	h.at(finale)
	assert.Equal(t, []moments.Command{
		closeDay(24),
		{Action: moments.OpenRound, Book: book, Round: dayRound(25), Times: 1, At: finale},
	}, h.execute(), "a day runs from 21:00 now, and the finale waits for the new end")

	h.at(seasonZeroEnds)
	assert.Empty(t, h.execute())

	h.at(finale.Add(7 * day))
	sent := h.execute()
	require.Len(t, sent, 8, "the days 25 to 31 close")
	assert.Equal(t, []moments.Command{
		closeDay(31),
		{Action: moments.OpenRound, Book: book, Round: battle, Times: 3, At: finale.Add(7 * day)},
	}, sent[6:])

	h.at(seasonZeroEnds.Add(7 * day))
	assert.Equal(t, []moments.Command{
		{Action: moments.CloseRound, Book: book, Round: battle, At: seasonZeroEnds.Add(7 * day)},
		{Action: moments.CloseBook, Book: book, Tiebreak: battle, At: seasonZeroEnds.Add(7 * day)},
	}, h.execute())
}

func TestADayThatOpenedRunsToASeasonEndMovedBeforeItsCutoff(t *testing.T) {
	h := newHarness(t, time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))
	h.noFinale()
	h.execute()
	end := dayEnds(13).Add(-2 * time.Hour)
	moved := zero
	moved.Length, moved.Finale = end.Sub(season.Origin), season.Finale{}
	require.NoError(t, h.seasons.Save(t.Context(), moved.Restore()))

	h.at(end.Add(-time.Second))
	assert.Empty(t, h.execute(), "day 13 runs on")

	h.at(end)
	assert.Equal(t, []moments.Command{
		{Action: moments.CloseRound, Book: book, Round: dayRound(13), At: end},
		{Action: moments.CloseBook, Book: book, At: end},
	}, h.execute(), "the day is cut at the season's end; with no finale, no round breaks a tie")

	h.boot()
	sent := h.execute()
	assert.Equal(t, moments.Command{Action: moments.CloseRound, Book: book, Round: dayRound(13), At: end}, sent[len(sent)-2],
		"a boot after it remembers the day opened")
}

func TestABattleEntryRunningWhenTheSeasonEndMovesIsCutAtTheNewEndAndIsNoFinale(t *testing.T) {
	h := newHarness(t, finale.Add(time.Minute))
	h.noFinale()
	h.execute()
	end := finale.Add(time.Hour)
	moved := zero
	moved.Length, moved.Finale = end.Sub(season.Origin), season.Finale{}
	require.NoError(t, h.seasons.Save(t.Context(), moved.Restore()))

	h.at(end)
	assert.Equal(t, []moments.Command{
		{Action: moments.CloseRound, Book: book, Round: moments.RoundID(rush.String()), At: end},
		{Action: moments.CloseBook, Book: book, At: end},
	}, h.execute())
}

func TestACommandRedisDidNotTakeIsSentAgainBeforeTheOnesAfterIt(t *testing.T) {
	h := newHarness(t, seasonZeroEnds.Add(-time.Second))
	h.execute()
	down := errors.New("redis is down")
	h.commands.err = down
	h.clock.Advance(time.Second)

	sent, err := h.useCase.Execute(t.Context())
	require.ErrorIs(t, err, down)
	assert.Empty(t, sent)

	h.commands.err = nil
	h.clock.Advance(time.Second)
	assert.Equal(t, []moments.Command{
		{Action: moments.CloseRound, Book: book, Round: battle, At: seasonZeroEnds},
		{Action: moments.CloseBook, Book: book, Tiebreak: battle, At: seasonZeroEnds},
	}, h.execute())
}

func TestNoSeasonStartedSendsNothing(t *testing.T) {
	h := newHarness(t, season.Origin.Add(-time.Second))

	assert.Empty(t, h.execute())
}

func TestSeasonsThatCannotBeReadSendNothing(t *testing.T) {
	h := newHarness(t, finale)
	down := errors.New("postgres is down")
	h.seasons.FailWith(down)

	_, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Empty(t, h.commands.sent)
}
