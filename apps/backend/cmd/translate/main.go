package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/raphoester/clickplanet.lol-backend/cmd/translate/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/translate/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/claude_translator"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_catalogues"
)

var errUntranslated = errors.New("a line was not translated")

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
	config, err := flags.Parse(args)
	if err != nil {
		return err //nolint:wrapcheck // the flags already say how to call the command.
	}
	work, err := page_catalogues.Load(ctx, config.Page, config.Languages)
	if err != nil {
		return fmt.Errorf("failed to read the page's catalogues: %w", err)
	}
	if config.Plan {
		return display.Plan(out, work) //nolint:wrapcheck // the writer's own error.
	}
	translator, err := translatorOf(config, work)
	if err != nil {
		return err
	}
	outcomes := page_catalogues.Translate(ctx, work, translator)
	done, err := display.Outcomes(out, config.Model, outcomes)
	if err != nil {
		return err //nolint:wrapcheck // the writer's own error.
	}
	if !done {
		return errUntranslated
	}
	return nil
}

// With nothing to send no key is needed: lines written by hand are fingerprinted all the same.
func translatorOf(config flags.Config, work page_catalogues.Work) (translation.Translator, error) {
	if work.ToSend() == 0 {
		return nothingToSend{}, nil
	}
	client, err := cpanthropic.NewClient(cpanthropic.Config{APIKey: config.APIKey})
	if err != nil {
		return nil, fmt.Errorf("%d lines to send: %w", work.ToSend(), err)
	}
	glossary, err := translation.ReadGlossary()
	if err != nil {
		return nil, err //nolint:wrapcheck // the glossary already says what does not read.
	}
	return claude_translator.New(client, config.Model, glossary), nil
}

var errAsked = errors.New("asked to translate with nothing to send")

type nothingToSend struct{}

func (nothingToSend) Translate(context.Context, translation.Language, []translation.Line) (map[translation.Key]string, translation.Spend, error) {
	return nil, translation.Spend{}, errAsked
}
