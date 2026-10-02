package cpconfigs

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrEnvVarNotFound = errors.New("environment variable not set")

const envScheme = "env://"

type EnvResolver struct{}

var _ SecretResolver = EnvResolver{}

func (EnvResolver) Resolve(identifier string) (string, error) {
	key, ok := strings.CutPrefix(identifier, envScheme)
	if !ok {
		return "", ErrNotEligible
	}

	value, found := os.LookupEnv(key)
	if !found {
		return "", fmt.Errorf("%w: %s", ErrEnvVarNotFound, key)
	}

	return value, nil
}
