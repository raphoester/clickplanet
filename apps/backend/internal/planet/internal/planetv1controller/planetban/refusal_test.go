package planetban_test

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetban"
)

func TestABannedCallerIsDeniedAndToldUntilWhen(t *testing.T) {
	until := time.Date(2026, 10, 21, 21, 40, 0, 0, time.UTC)

	refused, ok := planetban.Refusal(fmt.Errorf("wrapped: %w", clicks.NewBanRefusal(until)))
	require.True(t, ok)

	assert.Equal(t, connect.CodePermissionDenied, refused.Code())
	assert.NotContains(t, refused.Message(), "2026", "the time travels in the detail")
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	ban, isBan := detail.(*planetv1.BanRefusal)
	require.True(t, isBan)
	assert.Equal(t, until.UnixMilli(), ban.GetBannedUntilUnixMs())
}

func TestAnyOtherErrorIsNoBanRefusal(t *testing.T) {
	_, ok := planetban.Refusal(clicks.ErrThrottled)
	assert.False(t, ok)
}
