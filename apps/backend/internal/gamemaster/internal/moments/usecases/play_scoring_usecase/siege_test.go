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
	siegeEntry  = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000032"))
	siegeRound  = moments.BattleRoundOf(siegeEntry)
	siegeStarts = dayEnds(23)
	siegeEnds   = siegeStarts.Add(2 * time.Hour)
)

// A battle on Fri 30 Oct from 21:00 to 23:00: day 23 ends as it starts, and day 24 runs from its end.
func (h *harness) siege() {
	h.t.Helper()
	require.NoError(h.t, h.entries.Save(h.t.Context(), schedule.EntrySnapshot{
		ID: siegeEntry, Set: "capital-siege", Kind: schedule.Battle, Description: "Capital Siege.", StartsAt: siegeStarts,
		EndsAt: siegeEnds, Priority: 1, Times: 2, MadeAt: siegeStarts.Add(-day),
	}.Restore()))
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
	h := newHarness(t, siegeStarts.Add(-time.Second))
	h.siege()
	h.execute()

	h.at(siegeStarts)
	sent := h.execute()
	assert.Equal(t, []moments.Command{closeDay(23)}, sent, "the day closes; the battle waits for its judgement")

	h.judge(siegeRound, moments.Plain)
	h.at(siegeStarts.Add(time.Second))
	assert.Equal(t, []moments.Command{
		{Action: moments.OpenRound, Book: book, Round: siegeRound, Times: 2, At: siegeStarts},
	}, h.execute())
}

func TestASiegeCountsFromNothingAndTheDayAfterItWaitsForItsArena(t *testing.T) {
	h := newHarness(t, siegeStarts.Add(-time.Second))
	h.siege()
	h.judge(siegeRound, moments.Siege)
	h.execute()

	h.at(siegeStarts)
	assert.Equal(t, []moments.Command{
		closeDay(23),
		{Action: moments.OpenRound, Book: book, Round: siegeRound, Times: 2, At: siegeStarts, FromNothing: true},
	}, h.execute())

	h.at(siegeEnds)
	assert.Equal(t, []moments.Command{{Action: moments.CloseRound, Book: book, Round: siegeRound, At: siegeEnds}}, h.execute())
	h.at(siegeEnds.Add(5 * time.Second))
	assert.Empty(t, h.execute(), "day 24 waits for the map from before the siege")

	_, err := h.judged.KeepClosed(h.t.Context(), siegeRound)
	require.NoError(t, err)
	h.at(siegeEnds.Add(6 * time.Second))
	open, ok := opensRound(h.execute(), dayRound(24))
	require.True(t, ok)
	assert.Equal(t, moments.Command{Action: moments.OpenRound, Book: book, Round: dayRound(24), Times: 1, At: siegeEnds}, open)
}

func TestABootDuringTheWaitSendsTheCloseAgainAndHoldsTheDay(t *testing.T) {
	h := newHarness(t, siegeStarts.Add(-time.Second))
	h.siege()
	h.judge(siegeRound, moments.Siege)
	h.execute()
	h.at(siegeEnds.Add(time.Minute))
	h.execute()

	h.boot()
	sent := h.execute()

	assert.Contains(t, sent, moments.Command{Action: moments.CloseRound, Book: book, Round: siegeRound, At: siegeEnds})
	_, ok := opensRound(sent, dayRound(24))
	assert.False(t, ok)
}
