package uuid_id_provider_test

import (
	"testing"
	"time"

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

	assert.Equal(t, uuid.Version(7), uuid.UUID(first).Version())
	assert.NotEqual(t, first, second)
}

func TestAnIDSaysWhenTheAccountWasMadeForTheClickThrottle(t *testing.T) {
	before := time.Now().Truncate(time.Millisecond)
	id, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)

	created, ok := id.CreatedAt()
	require.True(t, ok)
	assert.False(t, created.Before(before))
	assert.False(t, created.After(time.Now()))
}
