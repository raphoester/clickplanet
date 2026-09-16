package uuid_id_provider_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/uuid_id_provider"
)

func TestIDsAreDistinctVersion7UUIDs(t *testing.T) {
	first, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)
	second, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)

	assert.Equal(t, uuid.Version(7), first.Version())
	assert.NotEqual(t, first, second)
}
