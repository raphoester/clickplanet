package cpconfigs

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

var (
	// ErrNotEligible is an identifier carrying another resolver's scheme, which is
	// how the chain moves on rather than fails.
	ErrNotEligible = errors.New("resolver not eligible for this identifier")

	// ErrSecretResolution is a resolver that matched the scheme and still could not
	// produce a value.
	ErrSecretResolution = errors.New("secret resolution failed")
)

// SecretResolver turns one identifier into the value behind it, returning
// ErrNotEligible and nothing else for a scheme that is not its own.
type SecretResolver interface {
	Resolve(identifier string) (string, error)
}

// resolveSecrets offers every incoming string to the chain, leaving alone the ones
// no resolver claims. A hook rather than a pass over the loaded map, so a field
// stays a plain string with no wrapper type and no tag.
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
