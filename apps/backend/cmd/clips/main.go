package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/clips/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

var (
	errUnsent         = errors.New("a clip was not scheduled")
	errUnknownCommand = errors.New("no such command")
)

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
	parsed, err := flags.Parse(args)
	if err != nil {
		return err //nolint:wrapcheck // the flags already say how to call the command.
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	return runCommand(ctx, parsed, out, logger)
}

func runCommand(ctx context.Context, parsed flags.Config, out io.Writer, logger *slog.Logger) error {
	switch parsed.Command {
	case flags.Fetch:
		return fetch(ctx, parsed, logger)
	case flags.Choose:
		return choose(ctx, parsed, out, logger)
	case flags.Schedule:
		return schedule(ctx, parsed, out, logger)
	default:
		return fmt.Errorf("%w: %q", errUnknownCommand, parsed.Command)
	}
}

func fetch(ctx context.Context, parsed flags.Config, logger *slog.Logger) error {
	if err := clips.Fetch(ctx, parsed.ReplayFile, parsed.RaceFile, parsed.SeasonFile, parsed.Job, logger); err != nil {
		return fmt.Errorf("failed to fetch the replay: %w", err)
	}
	return nil
}

func choose(ctx context.Context, parsed flags.Config, out io.Writer, logger *slog.Logger) error {
	picks, err := clips.Choose(ctx, parsed.ShortlistFile, parsed.Job, logger)
	if err != nil {
		return fmt.Errorf("failed to choose the clips of the run: %w", err)
	}
	return display.Picks(out, picks) //nolint:wrapcheck // the display already named what it could not write.
}

func schedule(ctx context.Context, parsed flags.Config, out io.Writer, logger *slog.Logger) error {
	report, err := clips.Schedule(ctx, parsed.ClipsFolder, parsed.Job, logger)
	if err != nil {
		return fmt.Errorf("failed to schedule the clips of the run: %w", err)
	}

	scheduled, err := display.Report(parsed.SummaryFile, out, report)
	if err != nil {
		return err //nolint:wrapcheck // the display already named what it could not write.
	}
	if !scheduled {
		return errUnsent
	}
	return nil
}
