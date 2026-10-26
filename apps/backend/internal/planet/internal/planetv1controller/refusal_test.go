package planetv1controller

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/refusal_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestARefusedCallerIsToldUntilWhenAndSpendsNoToken(t *testing.T) {
	at := time.Date(2026, 10, 21, 20, 40, 0, 0, time.UTC)
	until := at.Add(time.Hour)
	book := refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at))
	require.NoError(t, book.Decide(t.Context(), refusals.Caller{Scope: "1.2.3.4"}, refusals.Refused(until, at)))
	limiter := &fakeLimiter{allow: true}
	server := clickServerWith(t,
		refusal_click.New(throttle_click.New(stubService{}, limiter, onePrice, buckets), book, prometheus.NewRegistry()),
		nil, connect.WithInterceptors(errorNet()))

	_, err := clickAs(t, server, "1.2.3.4")

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	require.Len(t, connectErr.Details(), 1)
	detail, err := connectErr.Details()[0].Value()
	require.NoError(t, err)
	refusal, isBan := detail.(*planetv1.BanRefusal)
	require.True(t, isBan, "the detail is what tells a refusal from a VPN")
	assert.Equal(t, until.UnixMilli(), refusal.GetBannedUntilUnixMs())

	require.Equal(t, http.StatusForbidden, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, limiter.keys, "the refused caller never reached the bucket")

	_, err = clickAs(t, server, "5.6.7.8")
	require.NoError(t, err)
}
