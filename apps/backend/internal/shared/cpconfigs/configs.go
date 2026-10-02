package cpconfigs

import (
	"errors"
	"flag"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var ErrValidation = errors.New("config validation failed")

type Validator interface {
	Validate() error
}

type LoadOption func(loadParams) loadParams

type loadParams struct {
	path     string
	fromFlag bool
}

func FromFlag() LoadOption {
	return func(p loadParams) loadParams {
		p.fromFlag = true
		return p
	}
}

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

	if err := k.Load(env.Provider("", delimiter, nil), nil); err != nil {
		return fmt.Errorf("failed loading env variables: %w", err)
	}

	if err := k.UnmarshalWithConf("", config, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				resolveSecrets(resolvers),
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
			Result:           config,
			WeaklyTypedInput: true,
		},
	}); err != nil {
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

var resolvers = []SecretResolver{EnvResolver{}}

func pathFromFlag() string {
	path := flag.String("config", "", "path to config file")
	flag.Parse()

	return *path
}
