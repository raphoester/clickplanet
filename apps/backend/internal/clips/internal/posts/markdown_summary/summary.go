package markdown_summary

import (
	"fmt"
	"io"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

const timeLayout = "Mon 2 Jan, 15:04 UTC"

func New(out io.Writer) *Summary {
	return &Summary{out: out}
}

type Summary struct {
	out io.Writer
}

func (s *Summary) Write(report posts.Report) error {
	var page strings.Builder
	page.WriteString("| Clip | Goes out | |\n|---|---|---|\n")
	for _, outcome := range report {
		state := "scheduled"
		if outcome.Failure != "" {
			state = "**not scheduled**: " + cell(outcome.Failure)
		}
		fmt.Fprintf(&page, "| %s | %s | %s |\n", cell(outcome.Title), outcome.At.UTC().Format(timeLayout), state)
	}

	if _, err := io.WriteString(s.out, page.String()); err != nil {
		return fmt.Errorf("failed to write the summary: %w", err)
	}
	return nil
}

func cell(text string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(text)
}
