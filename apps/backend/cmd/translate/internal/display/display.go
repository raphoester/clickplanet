package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_catalogues"
)

// What each catalogue's languages would send, with nothing asked.
func Plan(out io.Writer, work page_catalogues.Work) error {
	var text strings.Builder
	for _, catalogue := range work.Catalogues {
		for _, plan := range work.Plans[catalogue.Name] {
			fmt.Fprintf(&text, "%s %s: %d to send, %d kept, %d written by hand, %d no longer in English\n",
				catalogue.Name, plan.Language, len(plan.Send), plan.Kept, plan.Adopted, plan.Dropped)
			for _, line := range plan.Send {
				fmt.Fprintf(&text, "  %s: %q\n", line.Key, line.English)
			}
		}
	}
	_, err := io.WriteString(out, text.String())
	return err //nolint:wrapcheck // the writer's own error.
}

// Says what came back, by catalogue and language, and the model's tokens; true when every line was translated.
func Outcomes(out io.Writer, model string, outcomes []page_catalogues.Outcome) (bool, error) {
	var (
		text  strings.Builder
		spend translation.Spend
	)
	done := true
	for _, outcome := range outcomes {
		spend = spend.Plus(outcome.Spend)
		fmt.Fprintf(&text, "%s %s: %d translated, %d kept, %d written by hand, %d no longer in English",
			outcome.Catalogue, outcome.Plan.Language, outcome.Translated, outcome.Plan.Kept, outcome.Plan.Adopted, outcome.Plan.Dropped)
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
	fmt.Fprintf(&text, "%s: %d tokens in, %d out\n", model, spend.InputTokens, spend.OutputTokens)
	if _, err := io.WriteString(out, text.String()); err != nil {
		return false, err //nolint:wrapcheck // the writer's own error.
	}
	return done, nil
}
