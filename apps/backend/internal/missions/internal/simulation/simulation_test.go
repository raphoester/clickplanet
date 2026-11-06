package simulation_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation"
)

var (
	start  = time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)
	played = []simulation.Named{{Name: "played", Sizing: flagmissions.Played}}
)

func dayOf(t *testing.T, season, n uint32, starts time.Time, length time.Duration) days.Day {
	t.Helper()

	day, err := days.DayOf(days.DaySnapshot{
		ID: days.DayID(fmt.Sprintf("season-%d/day-%d", season, n)), Season: season, Number: n, StartsAt: starts,
		EndsAt: starts.Add(length),
	})
	require.NoError(t, err)
	return day
}

// Russia, 5,000 tiles from 1, is the one ground big enough for a conquest; Sweden's 2,999 are not.
func conquestGlobe(t *testing.T) *simulation.Globe {
	t.Helper()

	grounds := make([]string, 0, 7_999)
	for n := range 7_999 {
		grounds = append(grounds, map[bool]string{true: "ru", false: "se"}[n < 5_000])
	}
	globe, err := simulation.NewGlobe(grounds)
	require.NoError(t, err)
	return globe
}

// Russia and Sweden, 2,000 tiles each, are too small for a conquest: a flag's one mission is a rival's shields.
func shieldGlobe(t *testing.T) *simulation.Globe {
	t.Helper()

	grounds := make([]string, 0, 4_000)
	for n := range 4_000 {
		grounds = append(grounds, map[bool]string{true: "ru", false: "se"}[n < 2_000])
	}
	globe, err := simulation.NewGlobe(grounds)
	require.NoError(t, err)
	return globe
}

type ledger struct {
	position uint64
	acts     []simulation.Act
}

func (l *ledger) takes(flag string, from, count uint32, at time.Time, account bool) {
	for n := range count {
		l.position++
		l.acts = append(l.acts, simulation.Take{
			Position: l.position, At: at.Add(time.Duration(n) * time.Second), Flag: flag, Tile: from + n, Account: account,
		})
	}
}

func (l *ledger) strikes(flag, owner string, tile, count uint32, at time.Time) {
	for n := range count {
		l.position++
		l.acts = append(l.acts, simulation.Strike{
			Position: l.position, At: at.Add(time.Duration(n) * time.Second), Flag: flag, Owner: owner, Tile: tile,
		})
	}
}

func rowsOf(t *testing.T, report simulation.Report, sizing int) []simulation.Row {
	t.Helper()

	require.Len(t, report.Sizings, len(played))
	return report.Sizings[sizing].Rows
}

func TestAConquestIsDrawnAtTheFlagsFirstTakeAndDoneByItsPlayersTakes(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 120, start.Add(time.Hour), true)

	report, err := simulation.Run([]days.Day{day}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	assert.Equal(t, []simulation.Row{{
		Sizing: "played", Day: "season-1/day-1", Flag: "fr", Share: 1.0 / 7_999, Kind: simulation.Conquest, Points: 10,
		Target: "ru", Goal: 100, Takes: 120, Progress: 100, Done: true, ToDone: 99 * time.Second,
	}}, rowsOf(t, report, 0), "the first day has no day before: the least goal; the biggest flag holds 900, France 1")
}

func TestTheDayBeforeSizesTheGoal(t *testing.T) {
	first, second := dayOf(t, 1, 1, start, 24*time.Hour), dayOf(t, 1, 2, start.Add(24*time.Hour), 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 400, start.Add(time.Hour), true)
	acts.takes("fr", 401, 1, start.Add(25*time.Hour), true)

	report, err := simulation.Run([]days.Day{first, second}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	rows := rowsOf(t, report, 0)
	require.Len(t, rows, 2)
	assert.Equal(t, uint64(400), rows[1].TakesBefore)
	assert.Equal(t, uint32(200), rows[1].Goal, "half of the 400 tiles of the day before")
	assert.Equal(t, uint32(1), rows[1].Progress)
	assert.False(t, rows[1].Done)
}

func TestABattleBetweenTwoDaysKeepsTheDayBefore(t *testing.T) {
	first, after := dayOf(t, 1, 1, start, 23*time.Hour), dayOf(t, 1, 2, start.Add(24*time.Hour), 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 400, start.Add(time.Hour), true)
	acts.takes("fr", 401, 50, start.Add(23*time.Hour+time.Minute), true)
	acts.takes("fr", 451, 1, start.Add(25*time.Hour), true)

	report, err := simulation.Run([]days.Day{first, after}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	rows := rowsOf(t, report, 0)
	require.Len(t, rows, 2)
	assert.Equal(t, uint64(400), rows[1].TakesBefore, "the battle's takes count in no day")
	assert.Equal(t, uint32(200), rows[1].Goal)
}

func TestASeasonsFirstDayStartsFromNothing(t *testing.T) {
	last, first := dayOf(t, 0, 24, start, 24*time.Hour), dayOf(t, 1, 1, start.Add(24*time.Hour), 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 400, start.Add(time.Hour), true)
	acts.takes("fr", 401, 1, start.Add(25*time.Hour), true)

	report, err := simulation.Run([]days.Day{last, first}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	rows := rowsOf(t, report, 0)
	require.Len(t, rows, 2)
	assert.Equal(t, uint64(0), rows[1].TakesBefore)
	assert.Equal(t, uint32(100), rows[1].Goal)
}

func TestATakeNobodySignedAndAFactBetweenTwoDaysCountNothingButPaintTheMap(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 50, start.Add(time.Hour), false)
	acts.takes("se", 101, 1, start.Add(2*time.Hour), true)
	acts.takes("de", 51, 50, start.Add(25*time.Hour), true)

	report, err := simulation.Run([]days.Day{day}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	rows := rowsOf(t, report, 0)
	require.Len(t, rows, 1, "neither France nor Germany had a fact that day")
	assert.Equal(t, "se", rows[0].Flag)
	assert.InDelta(t, 901.0/7_999, rows[0].Share, 1e-9)
}

func TestTheRivalsShieldsBrokenDoTheShieldMission(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	var acts ledger
	acts.takes("se", 2_001, 1_500, start.Add(-time.Hour), true)
	acts.strikes("fr", "fr", 1, 5, start.Add(time.Hour))
	acts.strikes("fr", "de", 1, 5, start.Add(2*time.Hour))
	acts.strikes("fr", "se", 2_001, 30, start.Add(3*time.Hour))

	report, err := simulation.Run([]days.Day{day}, acts.acts, shieldGlobe(t), played)

	require.NoError(t, err)
	assert.Equal(t, []simulation.Row{{
		Sizing: "played", Day: "season-1/day-1", Flag: "fr", Kind: simulation.Shields, Points: 10, Target: "se", Goal: 30,
		Strikes: 35, Progress: 30, Done: true, ToDone: 2*time.Hour + 29*time.Second,
	}}, rowsOf(t, report, 0), "its own shields are no shield broken, and Germany's are not Sweden's")
}

func TestEachSizingDrawsItsOwnMission(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 120, start.Add(time.Hour), true)
	harder := flagmissions.Sizing{GoalPercent: 50, LeastConquest: 300, LeastShields: 300, FewestPoints: 1, MostPoints: 4}

	report, err := simulation.Run([]days.Day{day}, acts.acts, conquestGlobe(t),
		append(played, simulation.Named{Name: "harder", Sizing: harder}))

	require.NoError(t, err)
	require.Len(t, report.Sizings, 2)
	assert.Equal(t, "harder", report.Sizings[1].Name)
	row := report.Sizings[1].Rows[0]
	assert.Equal(t, "harder", row.Sizing)
	assert.Equal(t, uint32(300), row.Goal)
	assert.Equal(t, uint32(4), row.Points)
	assert.Equal(t, uint32(120), row.Progress)
	assert.False(t, row.Done)
}

func TestTheLinesGroupTheFlagDaysBySizeAndKind(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	var acts ledger
	acts.takes("se", 5_001, 900, start.Add(-time.Hour), true)
	acts.takes("fr", 1, 200, start.Add(time.Hour), true)

	report, err := simulation.Run([]days.Day{day}, acts.acts, conquestGlobe(t), played)

	require.NoError(t, err)
	done := simulation.Line{
		FlagDays: 1, Done: 1, MedianToDone: 99 * time.Second, MedianGoal: 100, MedianGoalShare: 0.5,
		PointsOffered: 10, PointsWon: 10,
	}
	lines := map[string]simulation.Line{}
	for _, line := range report.Sizings[0].Lines {
		lines[line.Group] = line
	}
	for _, group := range []string{"flags under 0.5%", "conquests", "every flag"} {
		want := done
		want.Group = group
		assert.Equal(t, want, lines[group], group)
	}
	for _, group := range []string{"flags of 5% of the map or more", "flags of 0.5% to 5%", "shields to break"} {
		assert.Equal(t, simulation.Line{Group: group}, lines[group], group)
	}
}

func TestTheReplayCountsTheActsItCannotPlaceOnItsMap(t *testing.T) {
	day := dayOf(t, 1, 1, start, 24*time.Hour)
	acts := []simulation.Act{
		simulation.Take{Position: 1, At: start, Flag: "fr", Previous: "de", Tile: 1, Account: true},
		simulation.Clear{Position: 2, At: start, Previous: "fr", Tile: 1},
		simulation.Take{Position: 3, At: start, Flag: "fr", Tile: 8_000, Account: true},
	}

	report, err := simulation.Run([]days.Day{day}, acts, conquestGlobe(t), played)

	require.NoError(t, err)
	assert.Equal(t, uint64(1), report.Drift, "the ledger says Germany held tile 1, the replay nobody")
	assert.Equal(t, uint64(1), report.Outside)
}

func TestAMapWithATileOnNoLandIsRefused(t *testing.T) {
	_, err := simulation.NewGlobe([]string{"fr", ""})

	assert.ErrorIs(t, err, simulation.ErrNoGround)
}
