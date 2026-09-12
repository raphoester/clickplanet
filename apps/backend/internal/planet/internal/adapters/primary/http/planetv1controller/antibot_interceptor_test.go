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
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/ctxutil"
)

type fakeGuard struct {
	drop bool

	seen      []antibot.Click
	committed []antibot.Click
}

func (g *fakeGuard) Inspect(click antibot.Click) bool {
	g.seen = append(g.seen, click)
	return g.drop
}

func (g *fakeGuard) Committed(click antibot.Click) {
	g.committed = append(g.committed, click)
}

func (g *fakeGuard) Flagged() int { return len(g.seen) }

type fakeOwner map[uint32]string

func (o fakeOwner) Owner(tile uint32) (string, bool) {
	held, ok := o[tile]
	return held, ok
}

type clickRequest struct {
	connect.AnyRequest
	spec connect.Spec
	msg  *planetv1.ClickRequest
}

func (r clickRequest) Spec() connect.Spec { return r.spec }

func (r clickRequest) Any() any { return r.msg }

func antiBot(
	t *testing.T,
	ctx context.Context,
	guard ClickGuard,
	owner TileOwner,
	procedure string,
	msg *planetv1.ClickRequest,
) (bool, connect.AnyResponse, error) {
	t.Helper()

	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	clock := &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	interceptor, err := NewAntiBotInterceptor(guard, owner, clock, prometheus.NewRegistry())
	require.NoError(t, err)

	res, err := interceptor.WrapUnary(next)(ctx, clickRequest{spec: connect.Spec{Procedure: procedure}, msg: msg})

	return handlerRan, res, err
}

func TestAntiBotInterceptor(t *testing.T) {
	click := &planetv1.ClickRequest{TileId: 42, CountryId: "PS"}

	t.Run("lets an unflagged click reach the handler", func(t *testing.T) {
		guard := &fakeGuard{drop: false}

		ran, _, err := antiBot(t, context.Background(), guard, fakeOwner{}, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.True(t, ran)
		require.Len(t, guard.committed, 1, "a click the handler accepted reached the map")
	})

	t.Run("answers a flagged click OK without reaching the handler", func(t *testing.T) {
		guard := &fakeGuard{drop: true}

		ran, res, err := antiBot(t, context.Background(), guard, fakeOwner{}, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err, "a shadow ban must look exactly like success")
		require.False(t, ran, "the map must not be touched")
		require.IsType(t, &connect.Response[planetv1.ClickResponse]{}, res)
		require.Empty(t, guard.committed, "a dropped click took no tile")
	})

	t.Run("charges the same scope the throttle is charged to", func(t *testing.T) {
		guard := &fakeGuard{}

		ctx := ctxutil.AddIPToContext(context.Background(), "2001:db8::dead:beef")

		_, _, err := antiBot(t, ctx, guard, fakeOwner{}, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.Len(t, guard.seen, 1)
		assert.Equal(t, "2001:db8::/64", guard.seen[0].Scope)
	})

	t.Run("reads who held the tile before the handler could change it", func(t *testing.T) {
		guard := &fakeGuard{}

		_, _, err := antiBot(t, context.Background(), guard, fakeOwner{42: "FR"}, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.Len(t, guard.seen, 1)
		assert.Equal(t, "FR", guard.seen[0].Held)
		assert.False(t, guard.seen[0].NoOp, "PS taking a tile FR holds changes the map")
	})

	t.Run("marks a click onto a tile the caller's own country holds", func(t *testing.T) {
		guard := &fakeGuard{}

		_, _, err := antiBot(t, context.Background(), guard, fakeOwner{42: "PS"}, planetv1connect.ClickServiceClickProcedure, click)

		require.NoError(t, err)
		require.Len(t, guard.seen, 1)
		assert.True(t, guard.seen[0].NoOp, "it changes nothing and publishes nothing")
	})

	t.Run("ignores every procedure but Click", func(t *testing.T) {
		guard := &fakeGuard{drop: true}

		ran, _, err := antiBot(t, context.Background(), guard, fakeOwner{}, planetv1connect.ClickServiceGetMapProcedure, click)

		require.NoError(t, err)
		require.True(t, ran, "reads are never shadow banned")
		require.Empty(t, guard.seen)
	})
}

func TestAntiBotRunsAfterTheThrottle(t *testing.T) {
	guard := &fakeGuard{drop: true}

	clock := &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	interceptor, err := NewAntiBotInterceptor(guard, fakeOwner{}, clock, prometheus.NewRegistry())
	require.NoError(t, err)

	limiter := &fakeLimiter{allow: false}
	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		NewRateLimitInterceptor(limiter),
		interceptor,
	))

	require.Equal(t, http.StatusTooManyRequests, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, guard.seen, "a throttled click never reaches the guard")
}
