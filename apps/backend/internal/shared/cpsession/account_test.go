package cpsession_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestAnAccountIDSaysWhenItWasMade(t *testing.T) {
	at := time.Date(2026, 10, 1, 13, 37, 26, 938_000_000, time.UTC)

	created, ok := cpsession.AccountCreatedAt(at).CreatedAt()
	require.True(t, ok)
	assert.Equal(t, at, created)

	v7, err := uuid.NewV7()
	require.NoError(t, err)
	created, ok = cpsession.AccountID(v7).CreatedAt()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now(), created, time.Minute)
}

func TestAnIDThatIsNotAVersion7SaysNothing(t *testing.T) {
	for _, id := range []cpsession.AccountID{
		cpsession.NoAccount,
		cpsession.AccountID(uuid.New()),
		cpsession.AccountID(uuid.NewSHA1(uuid.NameSpaceOID, []byte("an-account"))),
	} {
		_, ok := id.CreatedAt()
		assert.False(t, ok, "%s", id)
	}
}
