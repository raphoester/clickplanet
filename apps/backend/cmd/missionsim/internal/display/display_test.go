package display_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/missionsim/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
)

var report = missions.SimulationReport{
	Drift: 3,
	Sizings: []missions.SizingReport{{
		Named: missions.NamedSizing{Name: "played", Sizing: missions.Sizing{
			GoalPercent: 50, LeastConquest: 100, LeastShields: 30, FewestPoints: 3, MostPoints: 10,
		}},
		Rows: []missions.SimulationRow{{
			Sizing: "played", Day: "season-1/day-3", Flag: "fr", Share: 0.004, Kind: "shields", Points: 9, Target: "se",
			Goal: 30, TakesBefore: 2_400, StrikesBefore: 40, Takes: 2_000, Strikes: 300, Progress: 30, Done: true,
			ToDone: 90 * time.Second,
		}},
		Lines: []missions.SimulationLine{
			{
				Group: "shields to break", FlagDays: 1, Done: 1, MedianToDone: 90 * time.Second, MedianGoal: 30,
				MedianGoalShare: 0.013, PointsOffered: 9, PointsWon: 9,
			},
			{Group: "flags of 5% of the map or more"},
		},
	}},
}

func TestTheReportIsATableASizing(t *testing.T) {
	var out bytes.Buffer

	require.NoError(t, display.Report(&out, report))

	text := out.String()
	assert.Contains(t, text, "played: goal 50% of the day before, at least 100 tiles or 30 shields, 3 to 10 points")
	assert.Regexp(t, `shields to break +1 +1 +100% +2 min +30 +1% +9 +9`, text)
	assert.Regexp(t, `flags of 5% of the map or more +0 +0 +- +- +0 +0% +0 +0`, text)
	assert.Contains(t, text, "3 acts found a tile wearing another flag than the replay said, 0 were past the map.")
}

func TestTheRowsAreACSVForASpreadsheet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rows.csv")

	require.NoError(t, display.Rows(path, report))

	written, err := os.ReadFile(path) //nolint:gosec // the test's own temporary file.
	require.NoError(t, err)
	assert.Equal(t, "sizing,day,flag,share,kind,points,target,goal,takes_before,strikes_before,takes,strikes,progress,"+
		"done,minutes_to_done\nplayed,season-1/day-3,fr,0.00400,shields,9,se,30,2400,40,2000,300,30,true,1.5\n",
		string(written))
}
