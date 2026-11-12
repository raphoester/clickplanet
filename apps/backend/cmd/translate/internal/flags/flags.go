package flags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/claude_translator"
)

const usage = `usage:
  translate [-page ../frontend] [-model claude-sonnet-5-5] [-languages fr,de] [-plan]

Brings the game page's catalogues up to its English (src/i18n/locales/en.json, the landmass names of
static/landmassNames.json, and "in" each flag of static/countries/countries.json, which English gets too): in each
language it has a file for, only the lines that are new, or whose English changed
since they were translated, are sent to Claude (ANTHROPIC_API_KEY), with the glossary and each line's context. A line
written by hand, with no fingerprint, is kept and fingerprinted. -plan says what would be sent and asks nothing. A new
language is an empty "{}" file named after its tag in src/i18n/locales.`

const KeyVariable = "ANTHROPIC_API_KEY"

var ErrArguments = errors.New("wrong arguments")

type Config struct {
	Page      string
	Model     string
	Languages []translation.Language
	// Says what would be sent, and sends nothing.
	Plan   bool
	APIKey string
}

func Parse(args []string) (Config, error) {
	set := flag.NewFlagSet("translate", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	page := set.String("page", "../frontend", "the game page's folder")
	model := set.String("model", claude_translator.DefaultModel, "the Claude model that translates")
	languages := set.String("languages", "", "the languages to bring up to date, all by default")
	plan := set.Bool("plan", false, "say what would be sent, and send nothing")
	if err := set.Parse(args); err != nil {
		return Config{}, fmt.Errorf("%w: %w\n%s", ErrArguments, err, usage)
	}
	if set.NArg() > 0 {
		return Config{}, fmt.Errorf("%w: %q\n%s", ErrArguments, set.Args(), usage)
	}
	config := Config{Page: *page, Model: *model, Plan: *plan, APIKey: os.Getenv(KeyVariable)}
	for _, language := range strings.Split(*languages, ",") {
		if language = strings.TrimSpace(language); language != "" {
			config.Languages = append(config.Languages, translation.Language(language))
		}
	}
	return config, nil
}
