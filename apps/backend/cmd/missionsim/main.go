package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/raphoester/clickplanet.lol-backend/cmd/missionsim/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/missionsim/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions"
)

var errUnknownCommand = errors.New("no such command")

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
	switch parsed.Command {
	case flags.Fetch:
		return fetch(ctx, parsed)
	case flags.Run:
		return simulate(parsed, out)
	default:
		return fmt.Errorf("%w: %q", errUnknownCommand, parsed.Command)
	}
}

func fetch(ctx context.Context, parsed flags.Config) error {
	if err := missions.FetchSeason(ctx, parsed.Host, parsed.Since, parsed.Folder); err != nil {
		return fmt.Errorf("failed to fetch the season: %w", err)
	}
	return nil
}

func simulate(parsed flags.Config, out io.Writer) error {
	report, err := missions.Simulate(parsed.Folder, parsed.Sizings)
	if err != nil {
		return fmt.Errorf("failed to simulate the season: %w", err)
	}
	if parsed.Rows != "" {
		if err := display.Rows(parsed.Rows, report); err != nil {
			return err //nolint:wrapcheck // the display already named what it could not write.
		}
	}
	return display.Report(out, report) //nolint:wrapcheck // the display already named what it could not write.
}
