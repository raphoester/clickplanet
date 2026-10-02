package cpconfigs

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

var (
	ErrNotEligible = errors.New("resolver not eligible for this identifier")

	ErrSecretResolution = errors.New("secret resolution failed")
)

type SecretResolver interface {
	Resolve(identifier string) (string, error)
}

func resolveSecrets(resolvers []SecretResolver) mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, _ reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}

		// Not a type assertion: a named string type has Kind String and would fail one.
		identifier := reflect.ValueOf(data).String()

		for _, resolver := range resolvers {
			value, err := resolver.Resolve(identifier)
			switch {
			case errors.Is(err, ErrNotEligible):
				continue
			case err != nil:
				return nil, fmt.Errorf("%w: %s: %w", ErrSecretResolution, identifier, err)
			default:
				return value, nil
			}
		}

		return data, nil
	}
}
