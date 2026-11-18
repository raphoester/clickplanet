package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_catalogues"
)

// What each catalogue's languages lack, and the batches written for them.
func Plan(out io.Writer, work page_catalogues.Work, batches []string) error {
	var text strings.Builder
	for _, catalogue := range work.Catalogues {
		for _, plan := range work.Plans[catalogue.Name] {
			fmt.Fprintf(&text, "%s %s: %d to translate, %d kept, %d written by hand, %d named on another board, %d no longer in English\n",
				catalogue.Name, plan.Language, len(plan.Send), plan.Kept, plan.Adopted, plan.Recalled, plan.Dropped)
		}
	}
	if len(work.MissingLayers) > 0 {
		fmt.Fprintf(&text, "Natural Earth: the page's cache lacks %s, so the landmass lines go without its names (npm run landmassNames downloads them)\n",
			strings.Join(work.MissingLayers, ", "))
	}
	for _, batch := range batches {
		fmt.Fprintf(&text, "wrote %s\n", batch)
	}
	_, err := io.WriteString(out, text.String())
	return err //nolint:wrapcheck // the writer's own error.
}

// What each catalogue's languages took from the answers; true when every line is translated.
func Outcomes(out io.Writer, outcomes []page_catalogues.Outcome) (bool, error) {
	var text strings.Builder
	done := true
	for _, outcome := range outcomes {
		fmt.Fprintf(&text, "%s %s: %d translated, %d kept, %d written by hand, %d named on another board, %d no longer in English",
			outcome.Catalogue, outcome.Plan.Language, outcome.Translated, outcome.Plan.Kept, outcome.Plan.Adopted,
			outcome.Plan.Recalled, outcome.Plan.Dropped)
		if outcome.Err != nil {
			done = false
			fmt.Fprintf(&text, ", failed: %s", outcome.Err)
		}
		text.WriteString("\n")
		for _, refusal := range outcome.Refused {
			done = false
			fmt.Fprintf(&text, "  refused %s: %s\n", refusal.Key, refusal.Reason)
		}
	}
	if _, err := io.WriteString(out, text.String()); err != nil {
		return false, err //nolint:wrapcheck // the writer's own error.
	}
	return done, nil
}
