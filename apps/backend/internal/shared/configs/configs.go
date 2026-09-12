// Package configs loads the process config: a YAML file, then the environment
// over it, then whatever the config says about itself.
//
// Where the file comes from is an option rather than the caller's business, so
// the binary asks for the config it wants and never for the flag, the parser or
// the precedence between the two.
package configs

import (
	"errors"
	"flag"
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ErrValidation is what a config's own Validate refused.
var ErrValidation = errors.New("config validation failed")

// Validator is a config that checks itself once it is loaded. Implementing it
// is optional, and it is the only place a bound can be refused with a sentence
// rather than a zero value nothing reports.
type Validator interface {
	Validate() error
}

type LoadOption func(loadParams) loadParams

type loadParams struct {
	path     string
	fromFlag bool
}

// FromFlag reads the path from -config, which is how the binary is run.
func FromFlag() LoadOption {
	return func(p loadParams) loadParams {
		p.fromFlag = true
		return p
	}
}

// Load fills config from the file, then the environment, then validates it.
//
// An empty path is not an error: every field keeps its zero value and the
// environment alone can carry a whole config, which is what the container does.
func Load(config any, opts ...LoadOption) error {
	var params loadParams
	for _, opt := range opts {
		params = opt(params)
	}

	if params.fromFlag {
		params.path = pathFromFlag()
	}

	k := koanf.New(delimiter)

	if params.path != "" {
		if err := k.Load(file.Provider(params.path), yaml.Parser()); err != nil {
			return fmt.Errorf("failed loading config file %q: %w", params.path, err)
		}
	}

	// Last loaded wins, so the environment overrides the file. The delimiter is
	// the nesting one, which is why tilesStorage.snapshotPath works as a name.
	if err := k.Load(env.Provider("", delimiter, nil), nil); err != nil {
		return fmt.Errorf("failed loading env variables: %w", err)
	}

	if err := k.Unmarshal("", config); err != nil {
		return fmt.Errorf("failed unmarshalling config: %w", err)
	}

	if validator, ok := config.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrValidation, err)
		}
	}

	return nil
}

const delimiter = "."

func pathFromFlag() string {
	path := flag.String("config", "", "path to config file")
	flag.Parse()

	return *path
}
