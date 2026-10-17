package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

const timeLayout = "Mon 2 Jan, 15:04 UTC"

// A JSON array on one line: the workflow makes one render job of each pick.
func Picks(out io.Writer, picks []clips.Pick) error {
	if err := json.NewEncoder(out).Encode(picks); err != nil {
		return fmt.Errorf("failed to write the picks: %w", err)
	}
	return nil
}

// Whether every clip of the report is scheduled on each of its platforms. The table is added to the summary file, or written to out when none is named.
func Report(summaryFile string, out io.Writer, report clips.Report) (bool, error) {
	if summaryFile == "" {
		return table(out, report)
	}

	file, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return false, fmt.Errorf("failed to open the summary: %w", err)
	}
	defer func() { _ = file.Close() }()

	return table(file, report)
}

func table(page io.Writer, report clips.Report) (bool, error) {
	var rows strings.Builder
	rows.WriteString("| Clip | Platform | Goes out | |\n|---|---|---|---|\n")
	for _, outcome := range report {
		for _, platform := range outcome.Platforms {
			state := "scheduled"
			if platform.Failure != "" {
				state = "**not scheduled**: " + cell(platform.Failure)
			}
			fmt.Fprintf(&rows, "| %s | %s | %s | %s |\n",
				cell(outcome.Title), platform.Platform, outcome.At.UTC().Format(timeLayout), state)
		}
	}

	if _, err := io.WriteString(page, rows.String()); err != nil {
		return false, fmt.Errorf("failed to write the summary: %w", err)
	}
	return report.Complete(), nil
}

func cell(text string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(text)
}
