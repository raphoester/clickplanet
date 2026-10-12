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

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

const usage = `usage:
  clips choose -shortlist <shortlist.json> [-count 2]
  clips schedule -clips <folder> [-review 4h] [-spacing 2h] [-summary <file>]

choose prints the picks of the stories to render, as a JSON array. schedule sends every clip of the folder to the
posting service, each for its time, and needs ZERNIO_API_KEY.`

const keyVariable = "ZERNIO_API_KEY"

var errArguments = errors.New("wrong arguments")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	switch args[0] {
	case "choose":
		return choose(ctx, args[1:], logger)
	case "schedule":
		return schedule(ctx, args[1:], logger)
	default:
		return fmt.Errorf("%w\n%s", errArguments, usage)
	}
}

func choose(ctx context.Context, args []string, logger *slog.Logger) error {
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

	if err := clips.Choose(ctx, *shortlist, config, os.Stdout, logger); err != nil {
		return fmt.Errorf("failed to choose the clips of the run: %w", err)
	}
	return nil
}

func schedule(ctx context.Context, args []string, logger *slog.Logger) error {
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

	page, closePage, err := summaryPage(*summary)
	if err != nil {
		return err
	}
	defer closePage()

	if err := clips.Schedule(ctx, *folder, config, page, logger); err != nil {
		return fmt.Errorf("failed to schedule the clips of the run: %w", err)
	}
	return nil
}

func summaryPage(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the file the operator of the job named.
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the summary: %w", err)
	}
	return file, func() { _ = file.Close() }, nil
}
