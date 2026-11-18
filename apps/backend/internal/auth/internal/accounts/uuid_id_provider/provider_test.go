package uuid_id_provider_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/uuid_id_provider"
)

func TestAnIDSaysWhenTheAccountWasMadeForTheClickThrottle(t *testing.T) {
	before := time.Now().Truncate(time.Millisecond)
	id, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)

	created, ok := id.CreatedAt()
	require.True(t, ok)
	assert.False(t, created.Before(before))
	assert.False(t, created.After(time.Now()))
}
