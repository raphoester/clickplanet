package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/migrations"
)

var errProblems = errors.New("the migrations would not run as written")

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	parsed, err := flags.Parse(args)
	if err != nil {
		return err //nolint:wrapcheck // the flags already say how to call the command.
	}

	problems, err := problems(ctx, parsed)
	if err != nil {
		return err
	}

	ok, err := display.Problems(out, problems)
	if err != nil {
		return err //nolint:wrapcheck // the display already named what it could not write.
	}
	if !ok {
		return errProblems
	}
	return nil
}

func problems(ctx context.Context, parsed flags.Config) ([]migrations.Problem, error) {
	head, err := migrations.TreeIn(os.DirFS("."))
	if err != nil {
		return nil, fmt.Errorf("failed to read the working tree: %w", err)
	}
	ran := migrations.Tree{}
	if parsed.Base != "" {
		ran, err = migrations.TreeAt(ctx, parsed.Base)
		if err != nil {
			return nil, fmt.Errorf("failed to read -base: %w", err)
		}
	}
	return migrations.Problems(ran, head), nil
}
