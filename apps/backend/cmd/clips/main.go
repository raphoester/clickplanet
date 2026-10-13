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

	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
)

var errUnsent = errors.New("a clip was not scheduled")

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
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	switch flags.Command {
	case fetchCommand:
		return fetchReplay(ctx, flags, logger)
	case chooseCommand:
		return chooseClips(ctx, flags, out, logger)
	default:
		return scheduleClips(ctx, flags, out, logger)
	}
}

func fetchReplay(ctx context.Context, flags FlagsConfig, logger *slog.Logger) error {
	if err := clips.Fetch(ctx, flags.ReplayFile, flags.Job, logger); err != nil {
		return fmt.Errorf("failed to fetch the replay: %w", err)
	}
	return nil
}

func chooseClips(ctx context.Context, flags FlagsConfig, out io.Writer, logger *slog.Logger) error {
	picks, err := clips.Choose(ctx, flags.ShortlistFile, flags.Job, logger)
	if err != nil {
		return fmt.Errorf("failed to choose the clips of the run: %w", err)
	}
	return displayPicks(out, picks)
}

func scheduleClips(ctx context.Context, flags FlagsConfig, out io.Writer, logger *slog.Logger) error {
	report, err := clips.Schedule(ctx, flags.ClipsFolder, flags.Job, logger)
	if err != nil {
		return fmt.Errorf("failed to schedule the clips of the run: %w", err)
	}

	page, closePage, err := openSummary(flags.SummaryFile, out)
	if err != nil {
		return err
	}
	defer closePage()

	scheduled, err := displayReport(page, report)
	if err != nil {
		return err
	}
	if !scheduled {
		return errUnsent
	}
	return nil
}
