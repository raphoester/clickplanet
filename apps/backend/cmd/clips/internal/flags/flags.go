package flags

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

const usage = `usage:
  clips fetch -out <replay.json> [-race <race.json>] [-season <season.json>] [-ssh <user@host>] [-hours 72] [-until <2026-10-12T12:00:00Z>]
  clips choose -findings <findings.json> [-count 2] [-platforms tiktok,youtube] [-summary <file>]
  clips edit -findings <findings.json> [-count 2] [-prompt]
  clips schedule -clips <folder> [-review 0] [-spacing 1h] [-platforms tiktok,youtube] [-sound-country US] [-summary <file>]
  clips stats [-ssh <user@host>] [-since 2026-10-13]

fetch asks the production backend's operator tool for a replay, over ssh: CLICKPLANET_SSH stands in for -ssh, and
the ledger keeps 72 hours in memory, so a replay cannot reach further back. With -race and -season it asks for the
season's race and calendar too, and leaves each out with a warning when the backend does not answer it. choose prints
the stories to render, as a JSON array, each with its number in the run and in an arm of every experiment that runs:
a day of the season won first, not told yet, then the stories Claude (ANTHROPIC_API_KEY) makes of the generator's
findings and headlines, each checked against them and against the headlines of the last week. It takes no more than
the platform with the most room left in its day takes: 14 posts in a trailing 24 hours on TikTok, 8 on YouTube.
schedule sends every clip of the folder to the posting service as one post on every platform with
room left, as soon as it can with no -review, and never sooner than -spacing after the last post on each; it refuses a
clip in an arm no experiment has. A clip the experiment "sound" drew for a trending track gets one of the top 20 of
TikTok's chart of -sound-country that play as long as it in the game's genres, the anthem muted, or goes out with
its anthem when the chart cannot be read or lists none that fit.
stats keeps a snapshot of the numbers of every post of the job scheduled since the
date, default the last 4 weeks, in the backend's postgres, over ssh as fetch asks: a row for each platform of each
post. All three need ZERNIO_API_KEY. A model that cannot answer leaves the run its day won alone; -summary adds how
the call went, and what it cost, to that file. edit tries the editor on saved findings, as many clips as -count, the
told stories read when ZERNIO_API_KEY is set: -prompt prints what it would be sent, counted when ANTHROPIC_API_KEY is
set, and asks for no answer.`

type Command string

const (
	Fetch    Command = "fetch"
	Choose   Command = "choose"
	Schedule Command = "schedule"
	Edit     Command = "edit"
	Stats    Command = "stats"
)

const (
	KeyVariable       = "ZERNIO_API_KEY"
	EditorKeyVariable = "ANTHROPIC_API_KEY"
	HostVariable      = "CLICKPLANET_SSH"
)

var ErrArguments = errors.New("wrong arguments")

type Config struct {
	Command Command

	ReplayFile string
	// Empty is no race.
	RaceFile string
	// Empty is no season.
	SeasonFile   string
	FindingsFile string
	ClipsFolder  string
	// Empty is the standard output for schedule, and none for choose, whose standard output is the picks.
	SummaryFile string
	// edit: print the prompt, ask for no answer.
	PromptOnly bool

	Job clips.Config
}

func Parse(args []string) (Config, error) {
	if len(args) == 0 {
		return Config{}, misused(ErrArguments)
	}

	switch Command(args[0]) {
	case Fetch:
		return parseFetch(args[1:])
	case Choose:
		return parseChoose(args[1:])
	case Schedule:
		return parseSchedule(args[1:])
	case Edit:
		return parseEdit(args[1:])
	case Stats:
		return parseStats(args[1:])
	default:
		return Config{}, misused(ErrArguments)
	}
}

func parseFetch(args []string) (Config, error) {
	parsed := Config{Command: Fetch}

	flags := flag.NewFlagSet(string(Fetch), flag.ContinueOnError)
	flags.StringVar(&parsed.ReplayFile, "out", "", "the file the replay is written to")
	flags.StringVar(&parsed.RaceFile, "race", "", "the file the season's race is written to, default none")
	flags.StringVar(&parsed.SeasonFile, "season", "", "the file the season's calendar is written to, default none")
	flags.StringVar(&parsed.Job.Replay.SSH, "ssh", os.Getenv(HostVariable), "user@host of the machine the backend runs on")
	hours := flags.Float64("hours", 0, "how long the replay is")
	until := flags.String("until", "", "when the replay ends, default now")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.ReplayFile == "" || parsed.Job.Replay.SSH == "" {
		return Config{}, misused(ErrArguments)
	}

	parsed.Job.Replay.Span = time.Duration(*hours * float64(time.Hour))
	if *until != "" {
		end, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			return Config{}, misused(err)
		}
		parsed.Job.Replay.Until = end
	}
	return parsed, nil
}

func parseChoose(args []string) (Config, error) {
	parsed := Config{Command: Choose}
	parsed.Job.Zernio.APIKey = os.Getenv(KeyVariable)
	parsed.Job.Anthropic.APIKey = os.Getenv(EditorKeyVariable)

	flags := flag.NewFlagSet(string(Choose), flag.ContinueOnError)
	flags.StringVar(&parsed.FindingsFile, "findings", "", "the file the generator's --findings wrote")
	flags.IntVar(&parsed.Job.PerRun, "count", 0, "how many clips a run schedules at most")
	flags.StringVar(&parsed.SummaryFile, "summary", "", "the file who chose the run is added to, default none")
	platforms := flags.String("platforms", "", "where each clip goes, default every platform")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.FindingsFile == "" {
		return Config{}, misused(ErrArguments)
	}
	return withPlatforms(parsed, *platforms)
}

func parseEdit(args []string) (Config, error) {
	parsed := Config{Command: Edit}
	parsed.Job.Zernio.APIKey = os.Getenv(KeyVariable)
	parsed.Job.Anthropic.APIKey = os.Getenv(EditorKeyVariable)

	flags := flag.NewFlagSet(string(Edit), flag.ContinueOnError)
	flags.StringVar(&parsed.FindingsFile, "findings", "", "the file the generator's --findings wrote")
	flags.IntVar(&parsed.Job.PerRun, "count", 0, "how many clips the run makes, default 2")
	flags.BoolVar(&parsed.PromptOnly, "prompt", false, "print what the model would be sent, and ask for no answer")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.FindingsFile == "" {
		return Config{}, misused(ErrArguments)
	}
	return parsed, nil
}

func parseSchedule(args []string) (Config, error) {
	parsed := Config{Command: Schedule}
	parsed.Job.Zernio.APIKey = os.Getenv(KeyVariable)

	flags := flag.NewFlagSet(string(Schedule), flag.ContinueOnError)
	flags.StringVar(&parsed.ClipsFolder, "clips", "", "the folder the clips were rendered in")
	flags.StringVar(&parsed.SummaryFile, "summary", "", "the file the summary of the run is added to, default the standard output")
	flags.DurationVar(&parsed.Job.Review, "review", 0, "how long after the run a clip goes out at the soonest, default none")
	flags.DurationVar(&parsed.Job.Spacing, "spacing", 0, "the least time between two posts on one platform, default 1h")
	platforms := flags.String("platforms", "", "where each clip goes, default every platform")
	country := flags.String("sound-country", "", "the country of TikTok's trending chart a trending track is picked from, default US")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.ClipsFolder == "" {
		return Config{}, misused(ErrArguments)
	}
	// Empty is the default, as the workflow passes an unset variable.
	if strings.TrimSpace(*country) != "" {
		code, err := clips.ChartCountryOf(*country)
		if err != nil {
			return Config{}, misused(err)
		}
		parsed.Job.SoundCountry = code
	}
	return withPlatforms(parsed, *platforms)
}

// Empty is every platform.
func withPlatforms(parsed Config, platforms string) (Config, error) {
	if names := strings.FieldsFunc(platforms, isSeparator); len(names) > 0 {
		chosen, err := clips.PlatformsOf(names...)
		if err != nil {
			return Config{}, misused(err)
		}
		parsed.Job.Platforms = chosen
	}
	return parsed, nil
}

func parseStats(args []string) (Config, error) {
	parsed := Config{Command: Stats}
	parsed.Job.Zernio.APIKey = os.Getenv(KeyVariable)

	flags := flag.NewFlagSet(string(Stats), flag.ContinueOnError)
	flags.StringVar(&parsed.Job.Replay.SSH, "ssh", os.Getenv(HostVariable), "user@host of the machine the backend runs on")
	since := flags.String("since", "", "the day of the first post read, default 4 weeks ago")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.Job.Replay.SSH == "" {
		return Config{}, misused(ErrArguments)
	}

	if *since != "" {
		day, err := time.Parse(time.DateOnly, *since)
		if err != nil {
			return Config{}, misused(err)
		}
		parsed.Job.StatsSince = day
	}
	return parsed, nil
}

func isSeparator(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}

func misused(cause error) error {
	return fmt.Errorf("%w\n%s", cause, usage)
}
