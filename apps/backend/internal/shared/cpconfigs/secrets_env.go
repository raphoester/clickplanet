package cpconfigs

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrEnvVarNotFound is an env:// anchor naming a variable that is not set.
var ErrEnvVarNotFound = errors.New("environment variable not set")

const envScheme = "env://"

// EnvResolver reads env://NAME from the environment, so the config file names the
// variable at the point the value is used instead of a table elsewhere doing it.
// An unset variable is an error: resolving it to "" would boot on a missing secret.
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
