package random_secret_generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/random_secret_generator"
)

func TestASecretIsAValidPKCEVerifierAndNeverRepeats(t *testing.T) {
	first, err := random_secret_generator.Generator{}.NewSecret()
	require.NoError(t, err)
	second, err := random_secret_generator.Generator{}.NewSecret()
	require.NoError(t, err)

	assert.Regexp(t, `^[A-Za-z0-9_-]{43}$`, first, "RFC 7636: 43 to 128 unreserved characters")
	assert.NotEqual(t, first, second)
}
