package main

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
func displayPicks(out io.Writer, picks []clips.Pick) error {
	if err := json.NewEncoder(out).Encode(picks); err != nil {
		return fmt.Errorf("failed to write the picks: %w", err)
	}
	return nil
}

// Whether every clip of the report is scheduled.
func displayReport(page io.Writer, report clips.Report) (bool, error) {
	var table strings.Builder
	table.WriteString("| Clip | Goes out | |\n|---|---|---|\n")
	for _, outcome := range report {
		state := "scheduled"
		if outcome.Failure != "" {
			state = "**not scheduled**: " + cell(outcome.Failure)
		}
		fmt.Fprintf(&table, "| %s | %s | %s |\n", cell(outcome.Title), outcome.At.UTC().Format(timeLayout), state)
	}

	if _, err := io.WriteString(page, table.String()); err != nil {
		return false, fmt.Errorf("failed to write the summary: %w", err)
	}
	return report.Complete(), nil
}

func cell(text string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(text)
}

func openSummary(path string, out io.Writer) (io.Writer, func(), error) {
	if path == "" {
		return out, func() {}, nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the summary: %w", err)
	}
	return file, func() { _ = file.Close() }, nil
}
