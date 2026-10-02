package random_code_generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/random_code_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestACodeIsSixDigitsAndDoesNotRepeat(t *testing.T) {
	seen := cpcolls.NewSet[string]()
	for range 50 {
		code, err := random_code_generator.Generator{}.NewCode()
		require.NoError(t, err)

		assert.Regexp(t, `^[0-9]{6}$`, code)
		seen.Add(code)
	}

	assert.Greater(t, seen.Len(), 45, "fifty draws from a million should almost never collide")
}
