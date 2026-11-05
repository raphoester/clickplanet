package aes_grant_sealer_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/aes_grant_sealer"
)

const token = "the-access-token"

var grant = grants.GrantID{15: 7}

func sealer(t *testing.T, seed byte) *aes_grant_sealer.Sealer {
	t.Helper()

	sealer, err := aes_grant_sealer.New(bytes.Repeat([]byte{seed}, 32))
	require.NoError(t, err)
	return sealer
}

func TestASealedTokenOpensAsItWas(t *testing.T) {
	s := sealer(t, 1)

	sealed, err := s.Sealed(grant, token)
	require.NoError(t, err)
	opened, err := s.Opened(grant, sealed)

	require.NoError(t, err)
	assert.Equal(t, token, opened)
}

func TestTheDatabaseCannotReadTheToken(t *testing.T) {
	sealed, err := sealer(t, 1).Sealed(grant, token)
	require.NoError(t, err)

	assert.NotContains(t, string(sealed), token)
}

func TestAChangedOrForeignTokenDoesNotOpen(t *testing.T) {
	sealed, err := sealer(t, 1).Sealed(grant, token)
	require.NoError(t, err)
	changed := bytes.Clone(sealed)
	changed[len(changed)-1] ^= 1
	foreign, err := sealer(t, 2).Sealed(grant, token)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		grant  grants.GrantID
		sealed []byte
	}{
		"changed":         {grant: grant, sealed: changed},
		"empty":           {grant: grant, sealed: nil},
		"another seed's":  {grant: grant, sealed: foreign},
		"another grant's": {grant: grants.GrantID{15: 8}, sealed: sealed},
		"cut short":       {grant: grant, sealed: sealed[:len(sealed)/2]},
	} {
		t.Run(name, func(t *testing.T) {
			opened, err := sealer(t, 1).Opened(tc.grant, tc.sealed)

			require.ErrorIs(t, err, aes_grant_sealer.ErrNotSealedHere)
			assert.Empty(t, opened)
		})
	}
}

func TestAnEmptySeedIsRefused(t *testing.T) {
	_, err := aes_grant_sealer.New(nil)

	assert.Error(t, err)
}
