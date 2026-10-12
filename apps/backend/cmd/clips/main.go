package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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
	keyVariable  = "ZERNIO_API_KEY"
	hostVariable = "CLICKPLANET_SSH"
)

var errArguments = errors.New("wrong arguments")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	switch args[0] {
	case "fetch":
		return fetch(ctx, args[1:], logger)
	case "choose":
		return choose(ctx, args[1:], out, logger)
	case "schedule":
		return schedule(ctx, args[1:], out, logger)
	default:
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
}

func fetch(ctx context.Context, args []string, logger *slog.Logger) error {
	flags := flag.NewFlagSet("fetch", flag.ContinueOnError)
	out := flags.String("out", "", "the file the replay is written to")
	hours := flags.Float64("hours", 0, "how long the replay is")
	var config clips.Config
	flags.StringVar(&config.Replay.SSH, "ssh", os.Getenv(hostVariable), "user@host of the machine the backend runs on")
	flags.Func("until", "when the replay ends, default now", func(value string) error {
		until, err := time.Parse(time.RFC3339, value)
		config.Replay.Until = until
		return err //nolint:wrapcheck // the flag set says which flag and which value.
	})
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w\n%s", err, usage)
	}
	if *out == "" || config.Replay.SSH == "" {
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
	config.Replay.Span = time.Duration(*hours * float64(time.Hour))

	if err := clips.Fetch(ctx, *out, config, logger); err != nil {
		return fmt.Errorf("failed to fetch the replay: %w", err)
	}
	return nil
}

func choose(ctx context.Context, args []string, out io.Writer, logger *slog.Logger) error {
	flags := flag.NewFlagSet("choose", flag.ContinueOnError)
	shortlist := flags.String("shortlist", "", "the file the generator's --shortlist wrote")
	var config clips.Config
	flags.IntVar(&config.PerRun, "count", 0, "how many clips a run schedules")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w\n%s", err, usage)
	}
	if *shortlist == "" {
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}

	if err := clips.Choose(ctx, *shortlist, config, out, logger); err != nil {
		return fmt.Errorf("failed to choose the clips of the run: %w", err)
	}
	return nil
}

func schedule(ctx context.Context, args []string, out io.Writer, logger *slog.Logger) error {
	flags := flag.NewFlagSet("schedule", flag.ContinueOnError)
	folder := flags.String("clips", "", "the folder the clips were rendered in")
	summary := flags.String("summary", "", "the file the summary of the run is added to, default the standard output")
	var config clips.Config
	flags.DurationVar(&config.Review, "review", 0, "how long before the first clip goes out")
	flags.DurationVar(&config.Spacing, "spacing", 0, "between two clips of one run")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w\n%s", err, usage)
	}
	if *folder == "" {
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
	config.Zernio.APIKey = os.Getenv(keyVariable)

	page, closePage, err := summaryPage(*summary, out)
	if err != nil {
		return err
	}
	defer closePage()

	if err := clips.Schedule(ctx, *folder, config, page, logger); err != nil {
		return fmt.Errorf("failed to schedule the clips of the run: %w", err)
	}
	return nil
}

func summaryPage(path string, out io.Writer) (io.Writer, func(), error) {
	if path == "" {
		return out, func() {}, nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the summary: %w", err)
	}
	return file, func() { _ = file.Close() }, nil
}
