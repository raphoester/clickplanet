package display

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
)

// One table a sizing, the played one first.
func Report(out io.Writer, report missions.SimulationReport) error {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
	for _, sizing := range report.Sizings {
		s := sizing.Sizing
		if _, err := fmt.Fprintf(writer, "\n%s: goal %d%% of the day before, at least %d tiles or %d shields, %d to %d points\n",
			sizing.Name, s.GoalPercent, s.LeastConquest, s.LeastShields, s.FewestPoints, s.MostPoints); err != nil {
			return fmt.Errorf("failed to write the report: %w", err)
		}
		if _, err := fmt.Fprintln(writer, "\tflag days\tdone\tdone %\tmedian time to done\tmedian goal\tgoal / the flag's day\t"+
			"points offered\tpoints won\t"); err != nil {
			return fmt.Errorf("failed to write the report: %w", err)
		}
		for _, line := range sizing.Lines {
			if _, err := fmt.Fprintf(writer, "%s\t%d\t%d\t%s\t%s\t%d\t%.0f%%\t%d\t%d\t\n", line.Group, line.FlagDays, line.Done,
				percent(line.Done, line.FlagDays), minutes(line.MedianToDone, line.Done), line.MedianGoal,
				100*line.MedianGoalShare, line.PointsOffered, line.PointsWon); err != nil {
				return fmt.Errorf("failed to write the report: %w", err)
			}
		}
	}
	if _, err := fmt.Fprintf(writer, "\n%d acts found a tile wearing another flag than the replay said, %d were past the map.\n",
		report.Drift, report.Outside); err != nil {
		return fmt.Errorf("failed to write the report: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write the report: %w", err)
	}
	return nil
}

func percent(part, whole int) string {
	if whole == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(part)/float64(whole))
}

func minutes(duration time.Duration, done int) string {
	if done == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f min", duration.Minutes())
}

// Every flag's day of every sizing, for a spreadsheet.
func Rows(path string, report missions.SimulationReport) error {
	file, err := os.Create(path) //nolint:gosec // the operator names the file.
	if err != nil {
		return fmt.Errorf("failed to make %s: %w", path, err)
	}
	writer := csv.NewWriter(file)
	_ = writer.Write([]string{
		"sizing", "day", "flag", "share", "kind", "points", "target", "goal", "takes_before", "strikes_before", "takes",
		"strikes", "progress", "done", "minutes_to_done",
	})
	for _, sizing := range report.Sizings {
		for _, row := range sizing.Rows {
			_ = writer.Write(recordOf(row))
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func recordOf(row missions.SimulationRow) []string {
	toDone := ""
	if row.Done {
		toDone = strconv.FormatFloat(row.ToDone.Minutes(), 'f', 1, 64)
	}
	return []string{
		row.Sizing, string(row.Day), row.Flag, strconv.FormatFloat(row.Share, 'f', 5, 64), string(row.Kind),
		strconv.FormatUint(uint64(row.Points), 10), row.Target, strconv.FormatUint(uint64(row.Goal), 10),
		strconv.FormatUint(row.TakesBefore, 10), strconv.FormatUint(row.StrikesBefore, 10),
		strconv.FormatUint(row.Takes, 10), strconv.FormatUint(row.Strikes, 10),
		strconv.FormatUint(uint64(row.Progress), 10), strconv.FormatBool(row.Done), toDone,
	}
}
