package flags

import (
	"errors"
	"flag"
	"fmt"
)

const usage = `usage:
  migrationcheck [-base <git revision>]

run from the backend's root. Checks each module's migrations: names golang-migrate reads, one migration per version,
an up file for each. With -base, a database that ran every migration of that revision: a migration added since must
come after the newest one it ran, and that newest one must still be there.`

var ErrArguments = errors.New("wrong arguments")

type Config struct {
	Base string
}

func Parse(args []string) (Config, error) {
	var parsed Config

	flags := flag.NewFlagSet("migrationcheck", flag.ContinueOnError)
	flags.StringVar(&parsed.Base, "base", "", "the git revision whose migrations the database already ran")
	if err := flags.Parse(args); err != nil {
		return Config{}, misused(err)
	}
	if flags.NArg() > 0 {
		return Config{}, misused(ErrArguments)
	}
	return parsed, nil
}

func misused(cause error) error {
	return fmt.Errorf("%w\n%s", cause, usage)
}
