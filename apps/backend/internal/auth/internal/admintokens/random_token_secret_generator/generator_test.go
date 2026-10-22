package random_token_secret_generator_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens/random_token_secret_generator"
)

func TestASecretIsItsPrefixAnd32RandomBytes(t *testing.T) {
	first, err := random_token_secret_generator.Generator{}.NewSecret(admintokens.AccessPrefix)
	require.NoError(t, err)
	second, err := random_token_secret_generator.Generator{}.NewSecret(admintokens.AccessPrefix)
	require.NoError(t, err)

	encoded, found := strings.CutPrefix(string(first), admintokens.AccessPrefix)
	require.True(t, found)
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
	assert.NotEqual(t, first, second)
}
