package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

const usage = `usage:
  clips fetch -out <replay.json> [-ssh <user@host>] [-hours 72] [-until <2026-10-12T12:00:00Z>]
  clips choose -shortlist <shortlist.json> [-count 2]
  clips schedule -clips <folder> [-review 4h] [-spacing 2h] [-summary <file>]

fetch asks the production backend's operator tool for a replay, over ssh: CLICKPLANET_SSH stands in for -ssh, and
the ledger keeps 72 hours in memory, so a replay cannot reach further back. choose prints the picks of the stories
to render, as a JSON array. schedule sends every clip of the folder to the posting service, each for its time, and
needs ZERNIO_API_KEY.`

const (
	fetchCommand    = "fetch"
	chooseCommand   = "choose"
	scheduleCommand = "schedule"
)

const (
	keyVariable  = "ZERNIO_API_KEY"
	hostVariable = "CLICKPLANET_SSH"
)

var errArguments = errors.New("wrong arguments")

type FlagsConfig struct {
	Command string

	ReplayFile    string
	ShortlistFile string
	ClipsFolder   string
	// Empty is the standard output.
	SummaryFile string

	Job clips.Config
}

func parseFlags(args []string) (FlagsConfig, error) {
	if len(args) == 0 {
		return FlagsConfig{}, misused(errArguments)
	}

	switch args[0] {
	case fetchCommand:
		return parseFetchFlags(args[1:])
	case chooseCommand:
		return parseChooseFlags(args[1:])
	case scheduleCommand:
		return parseScheduleFlags(args[1:])
	default:
		return FlagsConfig{}, misused(errArguments)
	}
}

func parseFetchFlags(args []string) (FlagsConfig, error) {
	parsed := FlagsConfig{Command: fetchCommand}

	flags := flag.NewFlagSet(fetchCommand, flag.ContinueOnError)
	flags.StringVar(&parsed.ReplayFile, "out", "", "the file the replay is written to")
	flags.StringVar(&parsed.Job.Replay.SSH, "ssh", os.Getenv(hostVariable), "user@host of the machine the backend runs on")
	hours := flags.Float64("hours", 0, "how long the replay is")
	until := flags.String("until", "", "when the replay ends, default now")
	if err := flags.Parse(args); err != nil {
		return FlagsConfig{}, misused(err)
	}
	if parsed.ReplayFile == "" || parsed.Job.Replay.SSH == "" {
		return FlagsConfig{}, misused(errArguments)
	}

	parsed.Job.Replay.Span = time.Duration(*hours * float64(time.Hour))
	if *until != "" {
		end, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			return FlagsConfig{}, misused(err)
		}
		parsed.Job.Replay.Until = end
	}
	return parsed, nil
}

func parseChooseFlags(args []string) (FlagsConfig, error) {
	parsed := FlagsConfig{Command: chooseCommand}

	flags := flag.NewFlagSet(chooseCommand, flag.ContinueOnError)
	flags.StringVar(&parsed.ShortlistFile, "shortlist", "", "the file the generator's --shortlist wrote")
	flags.IntVar(&parsed.Job.PerRun, "count", 0, "how many clips a run schedules")
	if err := flags.Parse(args); err != nil {
		return FlagsConfig{}, misused(err)
	}
	if parsed.ShortlistFile == "" {
		return FlagsConfig{}, misused(errArguments)
	}
	return parsed, nil
}

func parseScheduleFlags(args []string) (FlagsConfig, error) {
	parsed := FlagsConfig{Command: scheduleCommand}
	parsed.Job.Zernio.APIKey = os.Getenv(keyVariable)

	flags := flag.NewFlagSet(scheduleCommand, flag.ContinueOnError)
	flags.StringVar(&parsed.ClipsFolder, "clips", "", "the folder the clips were rendered in")
	flags.StringVar(&parsed.SummaryFile, "summary", "", "the file the summary of the run is added to, default the standard output")
	flags.DurationVar(&parsed.Job.Review, "review", 0, "how long before the first clip goes out")
	flags.DurationVar(&parsed.Job.Spacing, "spacing", 0, "between two clips of one run")
	if err := flags.Parse(args); err != nil {
		return FlagsConfig{}, misused(err)
	}
	if parsed.ClipsFolder == "" {
		return FlagsConfig{}, misused(errArguments)
	}
	return parsed, nil
}

func misused(cause error) error {
	return fmt.Errorf("%w\n%s", cause, usage)
}
