package planetv1controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
)

type fakeBlocklist struct {
	blocked map[string]ipblock.List
	asked   []string
}

func (b *fakeBlocklist) Blocked(ip string) (ipblock.List, bool) {
	b.asked = append(b.asked, ip)
	list, ok := b.blocked[ip]
	return list, ok
}

func vpnBlock(t *testing.T, ctx context.Context, blocklist ClickBlocklist, procedure string) (bool, prometheus.Gatherer, error) {
	t.Helper()

	registry := prometheus.NewRegistry()
	interceptor, err := NewVPNBlockInterceptor(blocklist, registry)
	require.NoError(t, err)

	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}}

	_, err = interceptor.WrapUnary(next)(ctx, req)

	return handlerRan, registry, err
}

func TestVPNBlockInterceptor(t *testing.T) {
	t.Run("refuses a click from a blocked address without reaching the handler", func(t *testing.T) {
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")
		blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"1.2.3.4": ipblock.ListVPN}}

		ran, registry, err := vpnBlock(t, ctx, blocklist, planetv1connect.ClickServiceClickProcedure)

		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		require.ErrorIs(t, err, ErrVPNBlocked)
		require.False(t, ran, "a refused click must not reach the domain")
		require.Equal(t, 1.0, testutil.ToFloat64(counter(t, registry, "vpn")))
	})

	t.Run("lets an address that is in no list through", func(t *testing.T) {
		ctx := ctxutil.AddIPToContext(context.Background(), "5.6.7.8")
		blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"1.2.3.4": ipblock.ListVPN}}

		ran, _, err := vpnBlock(t, ctx, blocklist, planetv1connect.ClickServiceClickProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("labels the refusal with the list that matched", func(t *testing.T) {
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")
		blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"1.2.3.4": ipblock.ListDatacenter}}

		_, registry, err := vpnBlock(t, ctx, blocklist, planetv1connect.ClickServiceClickProcedure)

		require.Error(t, err)
		require.Equal(t, 1.0, testutil.ToFloat64(counter(t, registry, "datacenter")))
	})

	t.Run("leaves the read procedures alone", func(t *testing.T) {
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")
		blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"1.2.3.4": ipblock.ListVPN}}

		for _, procedure := range []string{
			planetv1connect.ClickServiceGetMapProcedure,
			planetv1connect.ClickServiceMapDensityProcedure,
		} {
			ran, _, err := vpnBlock(t, ctx, blocklist, procedure)

			require.NoError(t, err, "a VPN user still loads and watches the planet")
			require.True(t, ran)
		}

		require.Empty(t, blocklist.asked, "reads are never even offered to the blocklist")
	})

	t.Run("lets a request with no source IP through", func(t *testing.T) {
		blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"": ipblock.ListVPN}}

		ran, _, err := vpnBlock(t, context.Background(), blocklist, planetv1connect.ClickServiceClickProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})
}

func TestVPNBlockOverHTTP(t *testing.T) {
	blocklist, err := ipblock.New(ipblock.Config{Enabled: true})
	require.NoError(t, err)

	blockInterceptor, err := NewVPNBlockInterceptor(blocklist, prometheus.NewRegistry())
	require.NoError(t, err)

	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		blockInterceptor,
		NewRateLimitInterceptor(allowAll{}),
	))

	require.Equal(t, http.StatusForbidden, clickStatus(t, server, "2.26.157.1"))
	require.Equal(t, http.StatusOK, clickStatus(t, server, "8.8.8.8"))
}

func TestVPNBlockRunsBeforeTheThrottle(t *testing.T) {
	blocklist := &fakeBlocklist{blocked: map[string]ipblock.List{"1.2.3.4": ipblock.ListVPN}}

	blockInterceptor, err := NewVPNBlockInterceptor(blocklist, prometheus.NewRegistry())
	require.NoError(t, err)

	limiter := &fakeLimiter{allow: true}
	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		blockInterceptor,
		NewRateLimitInterceptor(limiter),
	))

	require.Equal(t, http.StatusForbidden, clickStatus(t, server, "1.2.3.4"))
	require.Empty(t, limiter.keys, "the refused address never reached the bucket")
}

func clickServer(t *testing.T, options ...connect.HandlerOption) *httptest.Server {
	t.Helper()
	return clickServerReading(t, nil, options...)
}

func clickServerReading(
	t *testing.T,
	budgets ClickBudgetReader,
	options ...connect.HandlerOption,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		NewClickService(stubService{}, stubChecker{}, stubMapReader{}, stubSubscriber{}, budgets),
		options...,
	))

	server := httptest.NewServer(httpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server
}

type allowAll struct{}

func (allowAll) Take(string) (bool, ratelimit.State) { return true, ratelimit.State{} }

func counter(t *testing.T, gatherer prometheus.Gatherer, list string) prometheus.Counter {
	t.Helper()

	families, err := gatherer.Gather()
	require.NoError(t, err)

	found := prometheus.NewCounter(prometheus.CounterOpts{Name: "found"})
	for _, family := range families {
		if family.GetName() != "blocked_clicks" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "list" && label.GetValue() == list {
					found.Add(metric.GetCounter().GetValue())
				}
			}
		}
	}

	return found
}
