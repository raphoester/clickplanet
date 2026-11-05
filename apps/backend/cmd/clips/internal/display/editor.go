package display

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

// How the editor's call went, added to the summary file: what it cost and its reason, or why no answer was had.
// Nothing with no file named: choose's standard output is the picks, and its log says it too.
func Editorial(summaryFile string, editorial clips.Editorial) error {
	if summaryFile == "" {
		return nil
	}
	file, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return fmt.Errorf("failed to open the summary: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.WriteString(file, editorialOf(editorial)); err != nil {
		return fmt.Errorf("failed to write the summary: %w", err)
	}
	return nil
}

func editorialOf(editorial clips.Editorial) string {
	var page strings.Builder
	switch {
	case editorial.Failed != "":
		fmt.Fprintf(&page, "**Editor: no answer, the run is its day won alone.** %s.", oneLine(editorial.Failed))
	case editorial.Asked:
		page.WriteString("**Editor: the model.**")
		if editorial.Why != "" {
			fmt.Fprintf(&page, " %s.", oneLine(editorial.Why))
		}
	default:
		page.WriteString("**Editor: the model**, with nothing to decide: no call.")
	}
	if editorial.Asked {
		spend := editorial.Spend
		fmt.Fprintf(&page, " `%s`: %d tokens in (%d written to the cache, %d read from it), %d out, about $%.4f.",
			spend.Model, spend.InputTokens+spend.CacheWriteTokens+spend.CacheReadTokens, spend.CacheWriteTokens,
			spend.CacheReadTokens, spend.OutputTokens, spend.USD)
	}
	page.WriteString("\n\n")
	return page.String()
}

// The editor tried on findings: the prompt, and unless only that was asked for, what it chose, as the generator's
// --story takes each story, and what it cost.
func Trial(out io.Writer, trial clips.Trial, promptOnly bool) error {
	var page strings.Builder
	page.WriteString("=== instructions (cached) ===\n" + trial.Instructions + "\n=== the run ===\n" + trial.Prompt + "\n\n")
	if trial.Tokens > 0 {
		fmt.Fprintf(&page, "The prompt is %d tokens, about $%.5f in at most.\n", trial.Tokens,
			clips.SpendOf(trial.Tokens, 0).USD)
	} else {
		// About 4 characters a token, for English and JSON: a guess, which a key turns into a count.
		chars := len(trial.Instructions) + len(trial.Prompt)
		fmt.Fprintf(&page, "The prompt is %d characters, about %d tokens.\n", chars, chars/4)
	}
	if !trial.Offer.Open() {
		page.WriteString("Nothing for the model to decide: the run would ask nobody.\n")
	}
	if promptOnly {
		_, err := io.WriteString(out, page.String())
		return wrapped(err)
	}

	page.WriteString(editorialOf(trial.Choice.Editorial))
	for _, skipped := range trial.Choice.Skipped {
		fmt.Fprintf(&page, "skipped %s: %s\n", skipped.Story.Names(), oneLine(skipped.Why))
	}
	for _, chosen := range trial.Choice.Chosen {
		story, err := clips.StoryJSON(chosen.Story)
		if err != nil {
			return wrapped(err)
		}
		title := chosen.Story.Headline
		if chosen.Story.Seasonal > 0 {
			title = trial.Offer.Findings.Seasonal[chosen.Story.Seasonal-1].Title
		}
		fmt.Fprintf(&page, "#%d %s\n  --story '%s'\n", chosen.Pick, title, story)
	}
	_, err := io.WriteString(out, page.String())
	return wrapped(err)
}

func wrapped(err error) error {
	if err != nil {
		return fmt.Errorf("failed to write the trial: %w", err)
	}
	return nil
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(strings.TrimSuffix(strings.TrimSpace(text), ".")), " ")
}
