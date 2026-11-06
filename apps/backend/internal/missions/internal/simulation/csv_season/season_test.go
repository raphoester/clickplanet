package csv_season_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/csv_season"
)

var start = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)

func folderWith(t *testing.T, daysCSV, actsCSV string) string {
	t.Helper()

	folder := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(folder, csv_season.DaysFile), []byte(daysCSV), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(folder, csv_season.ActsFile), []byte(actsCSV), 0o600))
	return folder
}

func TestTheDaysAndTheActsAreReadInLedgerOrder(t *testing.T) {
	folder := folderWith(t,
		"season-1/day-1,1,1,1793512800000000,1793599200000000\n",
		"5,0,1793516400000000,clear,,se,7,t\n"+
			"4,1,1793516400000000,take,se,,10,t\n"+
			"4,0,1793516400000000,strike,se,de,5,t\n"+
			"4,0,1793516400000000,take,se,fr,4,t\n"+
			"2,0,1793516400000001,take,fr,,2,f\n")

	calendar, acts, err := csv_season.Read(folder)

	require.NoError(t, err)
	day, err := days.DayOf(days.DaySnapshot{
		ID: "season-1/day-1", Season: 1, Number: 1, StartsAt: start, EndsAt: start.Add(24 * time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, []days.Day{day}, calendar)
	at := start.Add(time.Hour)
	assert.Equal(t, []simulation.Act{
		simulation.Take{Position: 2, At: at.Add(time.Microsecond), Flag: "fr", Tile: 2},
		simulation.Take{Position: 4, At: at, Flag: "se", Previous: "fr", Tile: 4, Account: true},
		simulation.Take{Position: 4, Index: 1, At: at, Flag: "se", Tile: 10, Account: true},
		simulation.Strike{Position: 4, At: at, Flag: "se", Owner: "de", Tile: 5},
		simulation.Clear{Position: 5, At: at, Previous: "se", Tile: 7},
	}, acts)
}

func TestAnActTheDumpNeverWritesIsRefused(t *testing.T) {
	folder := folderWith(t, "", "1,0,1793516400000000,fortify,fr,,2,t\n")

	_, _, err := csv_season.Read(folder)

	assert.ErrorIs(t, err, csv_season.ErrUnknownAct)
}

func TestARowOfTheWrongWidthIsRefused(t *testing.T) {
	folder := folderWith(t, "season-1/day-1,1,1,1793512800000000\n", "")

	_, _, err := csv_season.Read(folder)

	assert.Error(t, err)
}
