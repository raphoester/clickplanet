package frozenmap_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/frozenmap"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

func TestAFrozenMapIsAFailedPreconditionThatSaysSo(t *testing.T) {
	refusal := frozenmap.Refusal(tempo.ErrFrozen)

	assert.Equal(t, connect.CodeFailedPrecondition, refusal.Code())
	require.Len(t, refusal.Details(), 1)
	detail, err := refusal.Details()[0].Value()
	require.NoError(t, err)
	assert.IsType(t, &planetv1.MapFrozen{}, detail)
}
