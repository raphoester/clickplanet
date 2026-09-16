package random_token_generator_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/random_token_generator"
)

func TestATokenIs32RandomBytesWithItsHash(t *testing.T) {
	first, err := random_token_generator.Generator{}.NewToken()
	require.NoError(t, err)
	second, err := random_token_generator.Generator{}.NewToken()
	require.NoError(t, err)

	raw, err := base64.RawURLEncoding.DecodeString(first.Value)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
	assert.NotEqual(t, first.Value, second.Value)
	assert.Equal(t, accounts.TokenOf(first.Value), first)
}
