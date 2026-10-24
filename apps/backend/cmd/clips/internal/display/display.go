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

// What the workflow's render job of a clip reads: its pick, and the arm of each experiment it is in, for the
// generator's --pick and --variants.
type assigned struct {
	Pick     clips.Pick     `json:"pick"`
	Variants clips.Variants `json:"variants"`
}

// A JSON array on one line, `[{"pick":1,"variants":{"length":"short"}}]`: the workflow makes one render job of each.
// Variants are `{}` for a clip in no experiment, never null.
func Picks(out io.Writer, assignments []clips.Assignment) error {
	picks := make([]assigned, 0, len(assignments))
	for _, assignment := range assignments {
		variants := assignment.Variants
		if variants == nil {
			variants = clips.Variants{}
		}
		picks = append(picks, assigned{Pick: assignment.Pick, Variants: variants})
	}
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
	rows.WriteString("| Clip | Arms | Platform | Goes out | |\n|---|---|---|---|---|\n")
	for _, outcome := range report {
		if outcome.Held {
			fmt.Fprintf(&rows, "| %s | %s | none | | held: every platform has posted its day |\n",
				cell(outcome.Title), armsOf(outcome.Variants))
			continue
		}
		for _, platform := range outcome.Platforms {
			state := "scheduled"
			if platform.Failure != "" {
				state = "**not scheduled**: " + cell(platform.Failure)
			}
			fmt.Fprintf(&rows, "| %s | %s | %s | %s | %s |\n",
				cell(outcome.Title), armsOf(outcome.Variants), platform.Platform, outcome.At.UTC().Format(timeLayout), state)
		}
	}

	if _, err := io.WriteString(page, rows.String()); err != nil {
		return false, fmt.Errorf("failed to write the summary: %w", err)
	}
	return report.Complete(), nil
}

// "length short, music anthem", or none for a clip in no experiment.
func armsOf(variants clips.Variants) string {
	if len(variants) == 0 {
		return "none"
	}
	return cell(variants.String())
}

func cell(text string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(text)
}
