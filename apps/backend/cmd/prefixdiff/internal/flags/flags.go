package flags

import (
	"errors"
	"flag"
	"fmt"
)

const usage = `usage:
  prefixdiff -keep <prefixes.txt> -remove <prefixes.txt>

prints every address of -keep that -remove does not hold, as the fewest prefixes, one a line. Each file holds one
prefix a line, as the vendored VPN lists do.`

var ErrArguments = errors.New("wrong arguments")

type Config struct {
	KeepFile   string
	RemoveFile string
}

func Parse(args []string) (Config, error) {
	var parsed Config

	flags := flag.NewFlagSet("prefixdiff", flag.ContinueOnError)
	flags.StringVar(&parsed.KeepFile, "keep", "", "the prefixes to print")
	flags.StringVar(&parsed.RemoveFile, "remove", "", "the prefixes to cut out of them")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if parsed.KeepFile == "" || parsed.RemoveFile == "" || flags.NArg() > 0 {
		return Config{}, misused(ErrArguments)
	}
	return parsed, nil
}

func misused(cause error) error {
	return fmt.Errorf("%w\n%s", cause, usage)
}
