package flags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

const usage = `usage:
  translations plan -work <folder> [-page ../frontend] [-languages fr,de] [-batch 400]
  translations apply -work <folder> [-page ../frontend] [-languages fr,de]

The translating is a Claude Code job (the repo's .claude/skills/translate); this command only does the bookkeeping,
and asks nothing of any network. plan reads the game page's catalogues (its lines, src/i18n/locales/en.json; the
landmass names of each board, static/boards/<board>/landmassNames-<hash>.json; "in" each flag of
static/countries/countries.json, which English gets too) and writes, in a new or empty -work folder, the lines each
language lacks or whose English changed since they were translated: <catalogue>/<language>.<n>.json, at most -batch
lines each. A translator writes its answers beside each batch, <language>.<n>.answers.json, a JSON object of each key
and its translation. apply writes every catalogue: the answers that keep their English's slots, with the fingerprint
of that English, and the lines kept; it fails on a line still not translated. Both keep a line written by hand, with
no fingerprint, and give a landmass the name another board gives the same English.`

type Command string

const (
	Plan  Command = "plan"
	Apply Command = "apply"
)

var (
	ErrArguments = errors.New("wrong arguments")
	errNoCommand = errors.New("no such command")
)

type Config struct {
	Command   Command
	Page      string
	Work      string
	Languages []translation.Language
	Batch     int
}

const defaultBatch = 400

func Parse(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{}, misused(nil)
	}
	config := Config{Command: Command(args[0])}
	set := flag.NewFlagSet(args[0], flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&config.Page, "page", "../frontend", "the game page's folder")
	set.StringVar(&config.Work, "work", "", "the folder of the batches and their answers")
	languages := set.String("languages", "", "the languages to bring up to date, all by default")
	switch config.Command {
	case Plan:
		set.IntVar(&config.Batch, "batch", defaultBatch, "the most lines in one batch")
	case Apply:
	default:
		return Config{}, misused(fmt.Errorf("%w: %q", errNoCommand, args[0]))
	}
	if err := set.Parse(args[1:]); err != nil {
		return Config{}, misused(err)
	}
	if set.NArg() > 0 || config.Work == "" || config.Command == Plan && config.Batch < 1 {
		return Config{}, misused(nil)
	}
	for language := range strings.SplitSeq(*languages, ",") {
		if language = strings.TrimSpace(language); language != "" {
			config.Languages = append(config.Languages, translation.Language(language))
		}
	}
	return config, nil
}

func misused(err error) error {
	return fmt.Errorf("%w\n%s", errors.Join(ErrArguments, err), usage)
}
