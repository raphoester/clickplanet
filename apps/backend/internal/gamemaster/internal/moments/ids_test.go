package moments_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

func TestTheIdsReadBackAsTheGamesWords(t *testing.T) {
	book := moments.BookOf(12)
	assert.Equal(t, moments.BookID("season-12"), book)
	number, err := moments.SeasonOf(book)
	require.NoError(t, err)
	assert.Equal(t, season.Number(12), number)

	day := moments.DayRoundOf(season.Day{Season: 0, Number: 24})
	assert.Equal(t, moments.RoundID("season-0/day-24"), day)
	named, err := moments.NamedOf(day)
	require.NoError(t, err)
	assert.Equal(t, moments.Named{Season: 0, Day: 24}, named)

	entry := schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))
	named, err = moments.NamedOf(moments.BattleRoundOf(entry))
	require.NoError(t, err)
	assert.Equal(t, moments.Named{Entry: entry}, named)
}

func TestAnIdTheGamemasterNeverGivesNamesNothing(t *testing.T) {
	for _, round := range []moments.RoundID{
		"", "season-0", "season-0/day-0", "season-01/day-1", "season-0/own-1",
		"0192F5A0-7C00-7000-8000-000000000031",
	} {
		_, err := moments.NamedOf(round)
		require.ErrorIs(t, err, moments.ErrUnknownRound, "%q", round)
	}
	for _, book := range []moments.BookID{"", "season-", "season-01", "season-0/day-1", "season-99999999999"} {
		_, err := moments.SeasonOf(book)
		require.ErrorIs(t, err, moments.ErrUnknownBook, "%q", book)
	}
}
