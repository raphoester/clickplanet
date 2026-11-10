package play_scoring_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

var (
	arenaEntry  = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000032"))
	arenaRound  = moments.BattleRoundOf(arenaEntry)
	arenaStarts = dayEnds(23)
	arenaEnds   = arenaStarts.Add(2 * time.Hour)
)

// A battle on Fri 30 Oct from 21:00 to 23:00: day 23 ends as it starts, and day 24 runs from its end.
func (h *harness) arenaBattle() {
	h.t.Helper()
	require.NoError(h.t, h.entries.Save(h.t.Context(), schedule.EntrySnapshot{
		ID: arenaEntry, Set: "arena-battle", Kind: schedule.Battle, Description: "Arena battle.", StartsAt: arenaStarts,
		EndsAt: arenaEnds, Priority: 1, Times: 2, MadeAt: arenaStarts.Add(-day),
	}.Restore()))
}

func (h *harness) arenaOpened() {
	h.t.Helper()
	_, err := h.judged.KeepOpened(h.t.Context(), arenaRound)
	require.NoError(h.t, err)
}

func opensRound(sent []moments.Command, round moments.RoundID) (moments.Command, bool) {
	for _, command := range sent {
		if command.Action == moments.OpenRound && command.Round == round {
			return command, true
		}
	}
	return moments.Command{}, false
}

func TestABattleOpensOnceItIsJudged(t *testing.T) {
	h := newHarness(t, arenaStarts.Add(-time.Second))
	h.arenaBattle()
	h.execute()

	h.at(arenaStarts)
	sent := h.execute()
	assert.Equal(t, []moments.Command{closeDay(23)}, sent, "the day closes; the battle waits for its judgement")

	h.judge(arenaRound, moments.Plain)
	h.at(arenaStarts.Add(time.Second))
	assert.Equal(t, []moments.Command{
		{Action: moments.OpenRound, Book: book, Round: arenaRound, Times: 2, At: arenaStarts, Ranking: moments.RankedByGain},
	}, h.execute())
}

func TestAnArenaBattleCountsFromNothingOnceItsArenaOpenedAndTheDayAfterItWaitsForItsArena(t *testing.T) {
	h := newHarness(t, arenaStarts.Add(-time.Second))
	h.arenaBattle()
	h.judge(arenaRound, moments.Arena)
	h.execute()

	h.at(arenaStarts)
	assert.Equal(t, []moments.Command{closeDay(23)}, h.execute(), "the battle waits for the blank map")

	h.arenaOpened()
	h.at(arenaStarts.Add(5 * time.Second))
	assert.Equal(t, []moments.Command{
		{Action: moments.OpenRound, Book: book, Round: arenaRound, Times: 2, At: arenaStarts, FromNothing: true, Ranking: moments.RankedByGain},
	}, h.execute())

	h.at(arenaEnds)
	assert.Equal(t, []moments.Command{{Action: moments.CloseRound, Book: book, Round: arenaRound, At: arenaEnds}}, h.execute())
	h.at(arenaEnds.Add(5 * time.Second))
	assert.Empty(t, h.execute(), "day 24 waits for the map from before the arena battle")

	_, err := h.judged.KeepClosed(h.t.Context(), arenaRound)
	require.NoError(t, err)
	h.at(arenaEnds.Add(6 * time.Second))
	open, ok := opensRound(h.execute(), dayRound(24))
	require.True(t, ok)
	assert.Equal(t, moments.Command{Action: moments.OpenRound, Book: book, Round: dayRound(24), Times: 1, At: arenaEnds, Ranking: moments.RankedByHeldAndGain}, open)
}

func TestABootDuringTheWaitHoldsTheDay(t *testing.T) {
	h := newHarness(t, arenaStarts.Add(-time.Second))
	h.arenaBattle()
	h.judge(arenaRound, moments.Arena)
	h.execute()
	h.at(arenaEnds.Add(time.Minute))
	h.execute()

	h.boot()
	sent := h.execute()

	assert.NotContains(t, sent, moments.Command{Action: moments.CloseRound, Book: book, Round: arenaRound, At: arenaEnds},
		"scoring answered it")
	_, ok := opensRound(sent, dayRound(24))
	assert.False(t, ok, "the arena is not closed")
}

func TestAnArenaBattleJudgedInItsLastSecondIsToldMissedAndTheDayAfterItWaitsForItsArenaAlone(t *testing.T) {
	h := newHarness(t, arenaStarts.Add(-time.Second))
	h.arenaBattle()
	h.execute()
	h.at(arenaStarts)
	h.execute()

	h.judge(arenaRound, moments.Arena)
	h.at(arenaEnds)
	assert.Equal(t, []moments.Command{{Action: moments.CloseRound, Book: book, Round: arenaRound, At: arenaEnds}}, h.execute())
	kept, err := h.opened.Kept(t.Context())
	require.NoError(t, err)
	assert.Contains(t, kept.MissedRounds, arenaRound)

	_, err = h.judged.KeepClosed(t.Context(), arenaRound)
	require.NoError(t, err)
	h.at(arenaEnds.Add(5 * time.Second))
	_, ok := opensRound(h.execute(), dayRound(24))
	assert.True(t, ok)
}

func TestAnArenaBattleWhoseArenaNeverOpensIsToldMissedAndCountsNothingOnTheWorldsMap(t *testing.T) {
	h := newHarness(t, arenaStarts.Add(-time.Second))
	h.arenaBattle()
	h.judge(arenaRound, moments.Arena)
	h.execute()

	for at := arenaStarts; at.Before(arenaEnds); at = at.Add(time.Minute) {
		h.at(at)
		_, ok := opensRound(h.execute(), arenaRound)
		require.False(t, ok, "at %s", at)
	}
	h.at(arenaEnds)
	assert.Equal(t, []moments.Command{{Action: moments.CloseRound, Book: book, Round: arenaRound, At: arenaEnds}}, h.execute())
	kept, err := h.opened.Kept(t.Context())
	require.NoError(t, err)
	assert.Contains(t, kept.MissedRounds, arenaRound)
	assert.NotContains(t, kept.Opened, arenaRound)
}
