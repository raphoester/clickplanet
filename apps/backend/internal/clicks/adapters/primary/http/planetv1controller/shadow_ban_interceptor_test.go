package planetv1controller

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
)

type observation struct {
	scope   string
	tile    uint32
	country string
}

type fakeShadowBanner struct {
	drop  bool
	seen  []observation
	takes []observation
}

func (b *fakeShadowBanner) Observe(scope string, tile uint32, country string) (bool, bool) {
	b.seen = append(b.seen, observation{scope: scope, tile: tile, country: country})
	return b.drop, !b.drop
}

func (b *fakeShadowBanner) Took(scope string, tile uint32) {
	b.takes = append(b.takes, observation{scope: scope, tile: tile})
}

func (b *fakeShadowBanner) Flagged() int { return len(b.seen) }

type clickRequest struct {
	connect.AnyRequest
	spec connect.Spec
	msg  *planetv1.ClickRequest
}

func (r clickRequest) Spec() connect.Spec { return r.spec }

func (r clickRequest) Any() any { return r.msg }

func shadowBan(
	t *testing.T,
	ctx context.Context,
	detector ClickShadowBanner,
	procedure string,
	msg *planetv1.ClickRequest,
) (bool, connect.AnyResponse, error) {
	t.Helper()

	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	interceptor, err := NewShadowBanInterceptor(detector, prometheus.NewRegistry())
	require.NoError(t, err)

	res, err := interceptor.WrapUnary(next)(ctx, clickRequest{spec: connect.Spec{Procedure: procedure}, msg: msg})

	return handlerRan, res, err
}

func TestShadowBanInterceptor(t *testing.T) {
	click := &planetv1.ClickRequest{TileId: 42, CountryId: "PS"}

	t.Run("lets an unflagged click reach the handler", func(t *testing.T) {
		detector := &fakeShadowBanner{drop: false}

		ran, _, err := shadowBan(t, context.Background(), detector, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("answers a flagged click OK without reaching the handler", func(t *testing.T) {
		detector := &fakeShadowBanner{drop: true}

		ran, res, err := shadowBan(t, context.Background(), detector, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err, "a shadow ban must look exactly like success")
		require.False(t, ran, "the map must not be touched")
		require.IsType(t, &connect.Response[planetv1.ClickResponse]{}, res)
	})

	t.Run("charges the same scope the throttle is charged to", func(t *testing.T) {
		detector := &fakeShadowBanner{}

		ctx := ctxutil.AddIPToContext(context.Background(), "2001:db8::dead:beef")

		_, _, err := shadowBan(t, ctx, detector, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.Equal(t, []observation{{scope: "2001:db8::/64", tile: 42, country: "PS"}}, detector.seen)
	})

	t.Run("ignores every procedure but Click", func(t *testing.T) {
		detector := &fakeShadowBanner{drop: true}

		ran, _, err := shadowBan(t, context.Background(), detector, planetv1connect.ClickServiceGetMapProcedure, click)

		require.NoError(t, err)
		require.True(t, ran, "reads are never shadow banned")
		require.Empty(t, detector.seen)
	})
}

func TestShadowBanRunsAfterTheThrottle(t *testing.T) {
	detector := &fakeShadowBanner{drop: true}

	interceptor, err := NewShadowBanInterceptor(detector, prometheus.NewRegistry())
	require.NoError(t, err)

	limiter := &fakeLimiter{allow: false}
	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		NewRateLimitInterceptor(limiter),
		interceptor,
	))

	// A shadow-banned caller keeps hitting the throttle like anyone else. Answer
	// it OK here and never being throttled is what gives the ban away.
	require.Equal(t, http.StatusTooManyRequests, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, detector.seen, "a throttled click never reaches the detector")
}
