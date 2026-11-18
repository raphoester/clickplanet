package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/cmd/translations/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/translations/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_catalogues"
)

var (
	errUntranslated   = errors.New("a line is not translated")
	errUnknownCommand = errors.New("no such command")
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	config, err := flags.Parse(args)
	if err != nil {
		return err //nolint:wrapcheck // the flags already say how to call the command.
	}
	work, err := page_catalogues.Load(config.Page, config.Languages)
	if err != nil {
		return fmt.Errorf("failed to read the page's catalogues: %w", err)
	}
	switch config.Command {
	case flags.Plan:
		return plan(config, work, out)
	case flags.Apply:
		return apply(config, work, out)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, config.Command)
	}
}

func plan(config flags.Config, work page_catalogues.Work, out io.Writer) error {
	batches, err := page_catalogues.WriteBatches(work, config.Work, config.Batch)
	if err != nil {
		return fmt.Errorf("failed to write the batches: %w", err)
	}
	return display.Plan(out, work, batches) //nolint:wrapcheck // the writer's own error.
}

func apply(config flags.Config, work page_catalogues.Work, out io.Writer) error {
	outcomes := page_catalogues.Apply(work, config.Work)
	done, err := display.Outcomes(out, outcomes)
	if err != nil {
		return err //nolint:wrapcheck // the writer's own error.
	}
	if !done {
		return errUntranslated
	}
	return nil
}
