package planetv1controller

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/ban_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
)

type stubBans map[string]time.Time

func (b stubBans) EnforcedSentence(_ context.Context, scope, _ string) (antibot.Sentence, bool) {
	until, ok := b[scope]
	return antibot.Sentence{Until: until}, ok
}

func TestABannedCallerIsToldUntilWhenAndSpendsNoToken(t *testing.T) {
	until := time.Date(2026, 10, 21, 21, 40, 0, 0, time.UTC)
	limiter := &fakeLimiter{allow: true}
	server := clickServerWith(t,
		ban_click.New(throttle_click.New(stubService{}, limiter, onePrice, buckets), stubBans{"1.2.3.4": until}, prometheus.NewRegistry()),
		nil, connect.WithInterceptors(errorNet()))

	_, err := clickAs(t, server, "1.2.3.4")

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	require.Len(t, connectErr.Details(), 1)
	detail, err := connectErr.Details()[0].Value()
	require.NoError(t, err)
	refusal, isBan := detail.(*planetv1.BanRefusal)
	require.True(t, isBan, "the detail is what tells a ban from a VPN")
	assert.Equal(t, until.UnixMilli(), refusal.GetBannedUntilUnixMs())

	require.Equal(t, http.StatusForbidden, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, limiter.keys, "the banned caller never reached the bucket")

	_, err = clickAs(t, server, "5.6.7.8")
	require.NoError(t, err)
}
