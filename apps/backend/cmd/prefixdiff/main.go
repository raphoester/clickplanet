package main

import (
	"fmt"
	"io"
	"net/netip"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/cmd/prefixdiff/internal/display"
	"github.com/raphoester/clickplanet.lol-backend/cmd/prefixdiff/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	parsed, err := flags.Parse(args)
	if err != nil {
		return err //nolint:wrapcheck // the flags already say how to call the command.
	}

	prefixes, err := difference(parsed)
	if err != nil {
		return err
	}

	return display.Prefixes(out, prefixes) //nolint:wrapcheck // the display already named what it could not write.
}

func difference(parsed flags.Config) ([]netip.Prefix, error) {
	keep, err := cpipblock.ParseFile(parsed.KeepFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read -keep: %w", err)
	}
	remove, err := cpipblock.ParseFile(parsed.RemoveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read -remove: %w", err)
	}
	return keep.Difference(remove).Prefixes(), nil
}
