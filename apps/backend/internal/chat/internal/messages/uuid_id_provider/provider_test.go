package uuid_id_provider_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/uuid_id_provider"
)

func TestIDsAreDistinctUUIDStrings(t *testing.T) {
	first, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)
	second, err := uuid_id_provider.Provider{}.NewID()
	require.NoError(t, err)

	parsed, err := uuid.Parse(string(first))
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(4), parsed.Version())
	assert.Equal(t, parsed.String(), string(first), "the canonical form chat.messages has always kept")
	assert.NotEqual(t, first, second)
}
