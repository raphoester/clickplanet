package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

// A round as the voice and the owner's page read it: each script with its id, what it says in one string, and how it
// measures (docs/video-lab.md, "The war report's bench").
type benchFile struct {
	At      time.Time    `json:"at"`
	USD     float64      `json:"usd"`
	Editor  benchEditor  `json:"editor"`
	Stories []benchStory `json:"stories"`
	Entries []benchEntry `json:"entries"`
}

type benchEditor struct {
	Why    string  `json:"why,omitempty"`
	Failed string  `json:"failed,omitempty"`
	USD    float64 `json:"usd"`
}

type benchStory struct {
	Story     int      `json:"story"`
	Of        string   `json:"of"`
	Kind      string   `json:"kind,omitempty"`
	Attacker  string   `json:"attacker,omitempty"`
	Rival     string   `json:"rival,omitempty"`
	Countries []string `json:"countries,omitempty"`
	Continent string   `json:"continent,omitempty"`
	Headline  string   `json:"headline"`
	Line      string   `json:"line,omitempty"`
}

type benchEntry struct {
	// "1-correspondent-opus": the story, the tone and the model, which names its audio too.
	ID      string   `json:"id"`
	Story   int      `json:"story"`
	Tone    string   `json:"tone"`
	Model   string   `json:"model"`
	Title   string   `json:"title,omitempty"`
	Lines   []string `json:"lines,omitempty"`
	Said    string   `json:"said,omitempty"`
	Why     string   `json:"why,omitempty"`
	Words   int      `json:"words"`
	Seconds float64  `json:"seconds"`
	Tells   []string `json:"tells"`
	USD     float64  `json:"usd"`
	Failed  string   `json:"failed,omitempty"`
}

// The round to its file, and a line a script to out: its id, its length, and what in it gives a machine away.
func Bench(file string, out io.Writer, round clips.BenchRound) error {
	written := benchFile{
		At: round.At.UTC(), USD: round.USD(),
		Editor: benchEditor{Why: round.Editorial.Why, Failed: round.Editorial.Failed, USD: round.Editorial.Spend.USD},
	}
	for i, story := range round.Stories {
		written.Stories = append(written.Stories, benchStory{
			Story: i + 1, Of: story.Names(), Kind: string(story.Kind), Attacker: story.Attacker, Rival: story.Rival,
			Countries: story.Countries, Continent: story.Continent, Headline: story.Headline, Line: story.Line,
		})
	}
	var page strings.Builder
	for _, entry := range round.Entries {
		script := entry.Script
		kept := benchEntry{
			ID: fmt.Sprintf("%d-%s-%s", entry.Story, entry.Tone, modelName(entry.Model)), Story: entry.Story,
			Tone: string(entry.Tone), Model: string(entry.Model), Title: script.Title, Lines: script.Lines,
			Said: script.Said(), Why: script.Why, Words: script.Words(), Seconds: roundedTo(script.Seconds(), 1),
			Tells: script.Tells(), USD: entry.Spend.USD, Failed: entry.Failed,
		}
		if kept.Tells == nil {
			kept.Tells = []string{}
		}
		written.Entries = append(written.Entries, kept)
		if kept.Failed != "" {
			fmt.Fprintf(&page, "%s: failed: %s\n", kept.ID, oneLine(kept.Failed))
			continue
		}
		fmt.Fprintf(&page, "%s: %d words, %.1f s, %d tells %v\n", kept.ID, kept.Words, kept.Seconds, len(kept.Tells), kept.Tells)
	}
	fmt.Fprintf(&page, "%d scripts of %d stories, about $%.3f\n", len(written.Entries), len(written.Stories), written.USD)

	text, err := json.MarshalIndent(written, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to write the round: %w", err)
	}
	if err := os.WriteFile(file, append(text, '\n'), 0o600); err != nil {
		return fmt.Errorf("failed to write the round: %w", err)
	}
	_, err = io.WriteString(out, page.String())
	return wrapped(err)
}

// "opus" for claude-opus-5-5.
func modelName(model clips.WriterModel) string {
	return strings.TrimSuffix(strings.TrimPrefix(string(model), "claude-"), "-5-5")
}

func roundedTo(value float64, places int) float64 {
	scale := 1.0
	for range places {
		scale *= 10
	}
	return float64(int(value*scale+0.5)) / scale
}
